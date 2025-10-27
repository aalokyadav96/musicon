package musicon

import (
	"context"
	"fmt"
	"naevis/db"
	"naevis/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --------------------------- Helpers ---------------------------

// fetchSongsByIDs retrieves songs by IDs, only published ones
func fetchSongsByIDs(ctx context.Context, ids []string) ([]Song, error) {
	if len(ids) == 0 {
		return []Song{}, nil
	}

	cursor, err := db.SongsCollection.Find(ctx, bson.M{"songid": bson.M{"$in": ids}, "published": true})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var songs []Song
	if err := cursor.All(ctx, &songs); err != nil {
		return nil, err
	}
	return songs, nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}, message string) {
	utils.RespondWithJSON(w, status, map[string]interface{}{
		"success": true,
		"data":    data,
		"message": message,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	utils.RespondWithJSON(w, status, map[string]interface{}{
		"success": false,
		"data":    nil,
		"message": message,
	})
}

func getPaginationParams(r *http.Request) (limit int64, page int64) {
	limit = 20
	page = 1
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.ParseInt(p, 10, 64); err == nil && parsed > 0 {
			page = parsed
		}
	}
	return
}

// --------------------------- Playlist Handlers ---------------------------

func GetUserPlaylists(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := utils.GetUserIDFromRequest(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cursor, err := db.PlaylistsCollection.Find(ctx, bson.M{"userid": userID})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch playlists")
		return
	}
	defer cursor.Close(ctx)

	var playlists []Playlist
	if err := cursor.All(ctx, &playlists); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode playlists")
		return
	}

	respondJSON(w, http.StatusOK, playlists, "Playlists fetched successfully")
}

func CreatePlaylist(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := utils.GetUserIDFromRequest(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
		return
	}

	type Req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	var req Req
	if err := utils.ParseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON input")
		return
	}

	if len(req.Name) == 0 || len(req.Name) > 100 {
		respondError(w, http.StatusBadRequest, "Playlist name must be 1-100 characters")
		return
	}

	newPlaylist := Playlist{
		Name:        req.Name,
		Description: req.Description,
		UserID:      userID,
		PlaylistID:  "pl_" + utils.GenerateRandomString(12),
		Songs:       []string{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Duration:    0,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := db.PlaylistsCollection.InsertOne(ctx, newPlaylist); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create playlist")
		return
	}

	respondJSON(w, http.StatusCreated, newPlaylist, "Playlist created successfully")
}

func DeletePlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userID := utils.GetUserIDFromRequest(r)
	playlistID := ps.ByName("playlistid")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res := db.PlaylistsCollection.FindOneAndDelete(ctx, bson.M{"playlistid": playlistID, "userid": userID})
	if res.Err() != nil {
		if res.Err() == mongo.ErrNoDocuments {
			respondError(w, http.StatusNotFound, "Playlist not found or unauthorized")
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to delete playlist")
		}
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID}, "Playlist deleted successfully")
}

func AddSongToPlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userID := utils.GetUserIDFromRequest(r)
	playlistID := ps.ByName("playlistid")
	songID := ps.ByName("songid")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	filter := bson.M{"playlistid": playlistID, "userid": userID}
	update := bson.M{"$addToSet": bson.M{"songs": songID}, "$set": bson.M{"updatedAt": time.Now()}}
	res, err := db.PlaylistsCollection.UpdateOne(ctx, filter, update)
	if err != nil || res.MatchedCount == 0 {
		respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID, "song_id": songID}, "Song added to playlist")
}

func RemoveSongFromPlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userID := utils.GetUserIDFromRequest(r)
	playlistID := ps.ByName("playlistid")
	songID := ps.ByName("songid")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	filter := bson.M{"playlistid": playlistID, "userid": userID}
	update := bson.M{"$pull": bson.M{"songs": songID}, "$set": bson.M{"updatedAt": time.Now()}}
	res, err := db.PlaylistsCollection.UpdateOne(ctx, filter, update)
	if err != nil || res.MatchedCount == 0 {
		respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID, "song_id": songID}, "Song removed from playlist")
}

func UpdatePlaylistInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	userID := utils.GetUserIDFromRequest(r)
	playlistID := ps.ByName("playlistid")

	type Req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CoverURL    string `json:"coverUrl"`
	}
	var req Req
	if err := utils.ParseJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON input")
		return
	}

	if len(req.Name) == 0 || len(req.Name) > 100 {
		respondError(w, http.StatusBadRequest, "Playlist name must be 1-100 characters")
		return
	}

	update := bson.M{"$set": bson.M{
		"name":        req.Name,
		"description": req.Description,
		"coverUrl":    req.CoverURL,
		"updatedAt":   time.Now(),
	}}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res, err := db.PlaylistsCollection.UpdateOne(ctx, bson.M{"playlistid": playlistID, "userid": userID}, update)
	if err != nil || res.MatchedCount == 0 {
		respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID}, "Playlist updated successfully")
}

// --------------------------- Albums & Songs ---------------------------

func GetAlbums(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cursor, err := db.AlbumsCollection.Find(ctx, bson.M{"published": true})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch albums")
		return
	}
	defer cursor.Close(ctx)

	var albums []Album
	if err := cursor.All(ctx, &albums); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode albums")
		return
	}

	respondJSON(w, http.StatusOK, albums, "Albums fetched successfully")
}

func GetAlbumSongs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	albumID := ps.ByName("albumid")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var album Album
	err := db.AlbumsCollection.FindOne(ctx, bson.M{"albumid": albumID}).Decode(&album)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			respondJSON(w, http.StatusOK, []Song{}, "No songs found for album")
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to fetch album")
		}
		return
	}

	songs, err := fetchSongsByIDs(ctx, album.Songs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch songs")
		return
	}

	respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for album %s fetched", albumID))
}

func GetPlaylistSongs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	playlistID := ps.ByName("playlistid")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var playlist Playlist
	err := db.PlaylistsCollection.FindOne(ctx, bson.M{"playlistid": playlistID}).Decode(&playlist)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			respondJSON(w, http.StatusOK, []Song{}, "Playlist not found")
		} else {
			respondError(w, http.StatusInternalServerError, "Failed to fetch playlist")
		}
		return
	}

	songs, err := fetchSongsByIDs(ctx, playlist.Songs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch songs")
		return
	}

	respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for playlist %s fetched", playlistID))
}

// --------------------------- Artist Songs ---------------------------

func GetArtistsSongs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	artistID := ps.ByName("artistid")
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit, page := getPaginationParams(r)
	skip := (page - 1) * limit

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"artistid": artistID}}},
		{{Key: "$unwind", Value: "$songs"}},
		{{Key: "$match", Value: bson.M{"songs.published": true}}},
		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$songs"}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := db.SongsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch artist songs")
		return
	}
	defer cursor.Close(ctx)

	var songs []Song
	if err := cursor.All(ctx, &songs); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode songs")
		return
	}

	respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for artist %s fetched", artistID))
}

// --------------------------- Recommendations ---------------------------

func GetRecommendedSongs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit, page := getPaginationParams(r)
	opts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)

	cursor, err := db.SongsCollection.Find(ctx, bson.M{"published": true}, opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch recommended songs")
		return
	}
	defer cursor.Close(ctx)

	var songs []Song
	if err := cursor.All(ctx, &songs); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode songs")
		return
	}

	respondJSON(w, http.StatusOK, songs, "Recommended songs fetched")
}

func GetRecommendedAlbums(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit, page := getPaginationParams(r)
	opts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)

	cursor, err := db.AlbumsCollection.Find(ctx, bson.M{"published": true}, opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch recommended albums")
		return
	}
	defer cursor.Close(ctx)

	var albums []Album
	if err := cursor.All(ctx, &albums); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode albums")
		return
	}

	respondJSON(w, http.StatusOK, albums, "Recommended albums fetched")
}

func GetRecommendations(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	basedOn := strings.ToLower(r.URL.Query().Get("based_on"))

	filter := bson.M{"published": true}
	switch basedOn {
	case "recently_played":
		filter["plays"] = bson.M{"$gt": 0}
	case "language_en":
		filter["language"] = "en"
	case "genre_pop":
		filter["genre"] = "Pop"
	}

	limit, page := getPaginationParams(r)
	opts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)

	cursor, err := db.SongsCollection.Find(ctx, filter, opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch recommendations")
		return
	}
	defer cursor.Close(ctx)

	var songs []Song
	if err := cursor.All(ctx, &songs); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode songs")
		return
	}

	respondJSON(w, http.StatusOK, songs, "Personalized recommendations fetched")
}

func GetUserLikes(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userID := utils.GetUserIDFromRequest(r)
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	cursor, err := db.LikesCollection.Find(ctx, bson.M{"userid": userID})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch likes")
		return
	}
	defer cursor.Close(ctx)

	var likes []Playlist
	if err := cursor.All(ctx, &likes); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decode likes")
		return
	}

	respondJSON(w, http.StatusOK, likes, "Likes fetched successfully")
}

// package musicon

// import (
// 	"context"
// 	"fmt"
// 	"naevis/db"
// 	"naevis/utils"
// 	"net/http"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/julienschmidt/httprouter"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.mongodb.org/mongo-driver/mongo/options"
// )

// // --------------------------- Helpers ---------------------------

// func fetchSongsByIDs(ctx context.Context, ids []string) ([]Song, error) {
// 	if len(ids) == 0 {
// 		return []Song{}, nil
// 	}

// 	cursor, err := db.SongsCollection.Find(ctx, bson.M{"songid": bson.M{"$in": ids}, "published": true})
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer cursor.Close(ctx)

// 	var songs []Song
// 	if err := cursor.All(ctx, &songs); err != nil {
// 		return nil, err
// 	}
// 	return songs, nil
// }

// func respondJSON(w http.ResponseWriter, status int, data interface{}, message string) {
// 	utils.RespondWithJSON(w, status, map[string]interface{}{
// 		"success": true,
// 		"data":    data,
// 		"message": message,
// 	})
// }

// func respondError(w http.ResponseWriter, status int, message string) {
// 	utils.RespondWithJSON(w, status, map[string]interface{}{
// 		"success": false,
// 		"data":    nil,
// 		"message": message,
// 	})
// }

// func getPaginationParams(r *http.Request) (limit int64, page int64) {
// 	limit = 20
// 	page = 1
// 	if l := r.URL.Query().Get("limit"); l != "" {
// 		if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 {
// 			limit = parsed
// 		}
// 	}
// 	if p := r.URL.Query().Get("page"); p != "" {
// 		if parsed, err := strconv.ParseInt(p, 10, 64); err == nil && parsed > 0 {
// 			page = parsed
// 		}
// 	}
// 	return
// }

// // --------------------------- Handlers ---------------------------

// func GetUserPlaylists(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 	userID := utils.GetUserIDFromRequest(r)
// 	if userID == "" {
// 		respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	cursor, err := db.PlaylistsCollection.Find(ctx, bson.M{"userid": userID})
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch playlists")
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	var playlists []Playlist
// 	if err := cursor.All(ctx, &playlists); err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to decode playlists")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, playlists, "Playlists fetched successfully")
// }

// func CreatePlaylist(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 	userID := utils.GetUserIDFromRequest(r)
// 	if userID == "" {
// 		respondError(w, http.StatusUnauthorized, "Unauthorized or missing user ID")
// 		return
// 	}

// 	type Req struct {
// 		Name        string `json:"name"`
// 		Description string `json:"description"`
// 	}
// 	var req Req
// 	if err := utils.ParseJSON(r, &req); err != nil {
// 		respondError(w, http.StatusBadRequest, "Invalid JSON input")
// 		return
// 	}

// 	if len(req.Name) == 0 || len(req.Name) > 100 {
// 		respondError(w, http.StatusBadRequest, "Playlist name must be 1-100 characters")
// 		return
// 	}

// 	newPlaylist := Playlist{
// 		Name:        req.Name,
// 		Description: req.Description,
// 		UserID:      userID,
// 		PlaylistID:  "pl_" + utils.GenerateRandomString(12),
// 		Songs:       []string{},
// 		CreatedAt:   time.Now(),
// 		UpdatedAt:   time.Now(),
// 		Duration:    0,
// 	}

// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	if _, err := db.PlaylistsCollection.InsertOne(ctx, newPlaylist); err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to create playlist")
// 		return
// 	}

// 	respondJSON(w, http.StatusCreated, newPlaylist, "Playlist created successfully")
// }

// func DeletePlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	userID := utils.GetUserIDFromRequest(r)
// 	playlistID := ps.ByName("playlistid")

// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	// Check ownership
// 	res := db.PlaylistsCollection.FindOneAndDelete(ctx, bson.M{"playlistid": playlistID, "userid": userID})
// 	if res.Err() != nil {
// 		if res.Err() == mongo.ErrNoDocuments {
// 			respondError(w, http.StatusNotFound, "Playlist not found or unauthorized")
// 		} else {
// 			respondError(w, http.StatusInternalServerError, "Failed to delete playlist")
// 		}
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID}, "Playlist deleted successfully")
// }

// func AddSongToPlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	userID := utils.GetUserIDFromRequest(r)
// 	playlistID := ps.ByName("playlistid")
// 	songID := ps.ByName("songid")

// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	// Verify ownership
// 	filter := bson.M{"playlistid": playlistID, "userid": userID}
// 	update := bson.M{"$addToSet": bson.M{"songs": songID}, "$set": bson.M{"updatedAt": time.Now()}}
// 	res, err := db.PlaylistsCollection.UpdateOne(ctx, filter, update)
// 	if err != nil || res.MatchedCount == 0 {
// 		respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID, "song_id": songID}, "Song added to playlist")
// }

// func RemoveSongFromPlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	userID := utils.GetUserIDFromRequest(r)
// 	playlistID := ps.ByName("playlistid")
// 	songID := ps.ByName("songid")

// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	filter := bson.M{"playlistid": playlistID, "userid": userID}
// 	update := bson.M{"$pull": bson.M{"songs": songID}, "$set": bson.M{"updatedAt": time.Now()}}
// 	res, err := db.PlaylistsCollection.UpdateOne(ctx, filter, update)
// 	if err != nil || res.MatchedCount == 0 {
// 		respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID, "song_id": songID}, "Song removed from playlist")
// }

// func UpdatePlaylistInfo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	userID := utils.GetUserIDFromRequest(r)
// 	playlistID := ps.ByName("playlistid")

// 	type Req struct {
// 		Name        string `json:"name"`
// 		Description string `json:"description"`
// 		CoverURL    string `json:"coverUrl"`
// 	}
// 	var req Req
// 	if err := utils.ParseJSON(r, &req); err != nil {
// 		respondError(w, http.StatusBadRequest, "Invalid JSON input")
// 		return
// 	}

// 	if len(req.Name) == 0 || len(req.Name) > 100 {
// 		respondError(w, http.StatusBadRequest, "Playlist name must be 1-100 characters")
// 		return
// 	}

// 	update := bson.M{"$set": bson.M{
// 		"name":        req.Name,
// 		"description": req.Description,
// 		"coverUrl":    req.CoverURL,
// 		"updatedAt":   time.Now(),
// 	}}

// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	res, err := db.PlaylistsCollection.UpdateOne(ctx, bson.M{"playlistid": playlistID, "userid": userID}, update)
// 	if err != nil || res.MatchedCount == 0 {
// 		respondError(w, http.StatusForbidden, "Playlist not found or unauthorized")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, map[string]string{"playlist_id": playlistID}, "Playlist updated successfully")
// }

// // --------------------------- Albums & Songs ---------------------------

// func GetAlbums(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	cursor, err := db.AlbumsCollection.Find(ctx, bson.M{"published": true})
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch albums")
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	var albums []Album
// 	if err := cursor.All(ctx, &albums); err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to decode albums")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, albums, "Albums fetched successfully")
// }

// func GetAlbumSongs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	albumID := ps.ByName("albumid")
// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	var album Album
// 	err := db.AlbumsCollection.FindOne(ctx, bson.M{"albumid": albumID}).Decode(&album)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			respondJSON(w, http.StatusOK, []Song{}, "No songs found for album")
// 		} else {
// 			respondError(w, http.StatusInternalServerError, "Failed to fetch album")
// 		}
// 		return
// 	}

// 	songs, err := fetchSongsByIDs(ctx, album.Songs)
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch songs")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for album %s fetched", albumID))
// }

// func GetPlaylistSongs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	playlistID := ps.ByName("playlistid")
// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	var playlist Playlist
// 	err := db.PlaylistsCollection.FindOne(ctx, bson.M{"playlistid": playlistID}).Decode(&playlist)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			respondJSON(w, http.StatusOK, []Song{}, "Playlist not found")
// 		} else {
// 			respondError(w, http.StatusInternalServerError, "Failed to fetch playlist")
// 		}
// 		return
// 	}

// 	songs, err := fetchSongsByIDs(ctx, playlist.Songs)
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch songs")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for playlist %s fetched", playlistID))
// }

// func GetArtistsSongs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	artistID := ps.ByName("artistid")
// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	limit, page := getPaginationParams(r)
// 	skip := (page - 1) * limit

// 	// Aggregation pipeline
// 	pipeline := mongo.Pipeline{
// 		{{Key: "$match", Value: bson.M{"artistid": artistID}}},
// 		{{Key: "$unwind", Value: "$songs"}},
// 		{{Key: "$match", Value: bson.M{"songs.published": true}}},
// 		{{Key: "$replaceRoot", Value: bson.M{"newRoot": "$songs"}}},
// 		{{Key: "$skip", Value: skip}},
// 		{{Key: "$limit", Value: limit}},
// 	}

// 	cursor, err := db.SongsCollection.Aggregate(ctx, pipeline)
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch artist songs")
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	var songs []Song
// 	if err := cursor.All(ctx, &songs); err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to decode songs")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, songs, fmt.Sprintf("Songs for artist %s fetched", artistID))
// }

// // --------------------------- Recommendations ---------------------------

// func GetRecommendedSongs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	limit, page := getPaginationParams(r)
// 	opts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)

// 	// cursor, err := db.SongsCollection.Find(ctx, bson.M{"published": true}, opts)
// 	cursor, err := db.SongsCollection.Find(ctx, bson.M{"published": true}, opts)
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch recommended songs")
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	var songs []Song
// 	if err := cursor.All(ctx, &songs); err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to decode songs")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, songs, "Recommended songs fetched")
// }

// func GetRecommendedAlbums(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	limit, page := getPaginationParams(r)
// 	opts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)

// 	cursor, err := db.AlbumsCollection.Find(ctx, bson.M{"published": true}, opts)
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch recommended albums")
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	var albums []Album
// 	if err := cursor.All(ctx, &albums); err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to decode albums")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, albums, "Recommended albums fetched")
// }

// func GetRecommendations(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
// 	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
// 	defer cancel()

// 	basedOn := strings.ToLower(r.URL.Query().Get("based_on"))

// 	filter := bson.M{"published": true}

// 	switch basedOn {
// 	case "recently_played":
// 		filter["plays"] = bson.M{"$gt": 0}
// 	case "language_en":
// 		filter["language"] = "en"
// 	case "genre_pop":
// 		filter["genre"] = "Pop"
// 	default:
// 		// no additional filter
// 	}

// 	limit, page := getPaginationParams(r)
// 	opts := options.Find().SetLimit(limit).SetSkip((page - 1) * limit)

// 	cursor, err := db.SongsCollection.Find(ctx, filter, opts)
// 	if err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to fetch recommendations")
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	var songs []Song
// 	if err := cursor.All(ctx, &songs); err != nil {
// 		respondError(w, http.StatusInternalServerError, "Failed to decode songs")
// 		return
// 	}

// 	respondJSON(w, http.StatusOK, songs, "Personalized recommendations fetched")
// }
