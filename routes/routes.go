package routes

import (
	"naevis/middleware"
	"naevis/musicon"
	"naevis/ratelim"

	"github.com/julienschmidt/httprouter"
)

func AddMusicRoutes(router *httprouter.Router, rateLimiter *ratelim.RateLimiter) {
	// --------------------------- PLAYLISTS ---------------------------
	router.GET("/api/v1/musicon/user/playlists", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetUserPlaylists)))
	router.GET("/api/v1/musicon/user/liked", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetUserLikes)))
	router.POST("/api/v1/musicon/playlists", rateLimiter.Limit(middleware.Authenticate(musicon.CreatePlaylist)))
	router.DELETE("/api/v1/musicon/playlists/:playlistid", rateLimiter.Limit(middleware.Authenticate(musicon.DeletePlaylist)))

	// Add / Remove songs to playlist
	// router.POST("/api/v1/musicon/playlists/:playlistid/songs/:songid", rateLimiter.Limit(middleware.Authenticate(musicon.AddSongToPlaylist)))
	router.POST("/api/v1/musicon/playlists/:playlistid/songs", rateLimiter.Limit(middleware.Authenticate(musicon.AddSongToPlaylist)))
	router.POST("/api/v1/musicon/user/liked/:songid", rateLimiter.Limit(middleware.OptionalAuth(musicon.SetUserLikes)))
	router.DELETE("/api/v1/musicon/playlists/:playlistid/songs/:songid", rateLimiter.Limit(middleware.Authenticate(musicon.RemoveSongFromPlaylist)))

	// Playlist details
	router.GET("/api/v1/musicon/playlists/:playlistid/songs", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetPlaylistSongs)))

	// Rename / Update playlist info
	router.PATCH("/api/v1/musicon/playlists/:playlistid", rateLimiter.Limit(middleware.Authenticate(musicon.UpdatePlaylistInfo)))

	// --------------------------- ARTISTS ---------------------------
	router.GET("/api/v1/musicon/artists/:artistid/songs", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetArtistsSongs)))

	// --------------------------- ALBUMS ---------------------------
	router.GET("/api/v1/musicon/albums", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetAlbums)))
	router.GET("/api/v1/musicon/albums/:albumid/songs", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetAlbumSongs)))
	router.GET("/api/v1/musicon/recommended/albums", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendedAlbums)))

	// --------------------------- SONGS & RECOMMENDATIONS ---------------------------
	router.GET("/api/v1/musicon/recommended", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendedSongs)))

	// Dynamic personalized recommendations
	router.GET("/api/v1/musicon/recommendations", rateLimiter.Limit(middleware.OptionalAuth(musicon.GetRecommendations)))
}

// func AddMusicRoutes(router *httprouter.Router, rateLimiter *ratelim.RateLimiter) {
// 	// --------------------------- PLAYLISTS ---------------------------
// 	router.GET("/api/v1/musicon/user/playlists", middleware.OptionalAuth(musicon.GetUserPlaylists))
// 	router.POST("/api/v1/musicon/playlists", middleware.Authenticate(musicon.CreatePlaylist))
// 	router.DELETE("/api/v1/musicon/playlists/:playlistid", middleware.Authenticate(musicon.DeletePlaylist))

// 	// Add / Remove songs to playlist
// 	router.POST("/api/v1/musicon/playlists/:playlistid/songs/:songid", middleware.Authenticate(musicon.AddSongToPlaylist))
// 	router.DELETE("/api/v1/musicon/playlists/:playlistid/songs/:songid", middleware.Authenticate(musicon.RemoveSongFromPlaylist))

// 	// Playlist details
// 	router.GET("/api/v1/musicon/playlists/:playlistid/songs", middleware.OptionalAuth(musicon.GetPlaylistSongs))

// 	// Rename / Update playlist info (Phase 1 enhancement)
// 	router.PATCH("/api/v1/musicon/playlists/:playlistid", middleware.Authenticate(musicon.UpdatePlaylistInfo))

// 	// --------------------------- ARTISTS ---------------------------
// 	router.GET("/api/v1/musicon/artists/:artistid/songs", middleware.OptionalAuth(musicon.GetArtistsSongs))

// 	// --------------------------- ALBUMS ---------------------------
// 	router.GET("/api/v1/musicon/albums", middleware.OptionalAuth(musicon.GetAlbums))
// 	router.GET("/api/v1/musicon/albums/:albumid/songs", middleware.OptionalAuth(musicon.GetAlbumSongs))
// 	router.GET("/api/v1/musicon/recommended/albums", middleware.OptionalAuth(musicon.GetRecommendedAlbums))

// 	// --------------------------- SONGS & RECOMMENDATIONS ---------------------------
// 	router.GET("/api/v1/musicon/recommended", middleware.OptionalAuth(musicon.GetRecommendedSongs))

// 	// Dynamic personalized recommendations (Phase 2 enhancement)
// 	// Example: /api/v1/musicon/recommendations?based_on=recently_played
// 	router.GET("/api/v1/musicon/recommendations", middleware.OptionalAuth(musicon.GetRecommendations))
// }

// // func AddMusicRoutes(router *httprouter.Router, rateLimiter *ratelim.RateLimiter) {
// // 	router.GET("/api/v1/musicon/user/playlists", middleware.OptionalAuth(musicon.GetUserPlaylists))
// // 	router.POST("/api/v1/musicon/playlists", middleware.Authenticate(musicon.CreatePlaylist))
// // 	router.DELETE("/api/v1/musicon/playlists/:playlistid", middleware.Authenticate(musicon.DeletePlaylist))
// // 	router.POST("/api/v1/musicon/playlists/:playlistid/songs/:songid", middleware.Authenticate(musicon.AddSongToPlaylist))
// // 	router.DELETE("/api/v1/musicon/playlists/:playlistid/songs/:songid", middleware.Authenticate(musicon.RemoveSongFromPlaylist))

// // 	router.GET("/api/v1/musicon/artists/:artistid/songs", middleware.OptionalAuth(musicon.GetArtistsSongs))
// // 	router.GET("/api/v1/musicon/albums", middleware.OptionalAuth(musicon.GetAlbums))
// // 	router.GET("/api/v1/musicon/albums/:albumid/songs", middleware.OptionalAuth(musicon.GetAlbumSongs))
// // 	router.GET("/api/v1/musicon/recommended", middleware.OptionalAuth(musicon.GetRecommendedSongs))
// // 	router.GET("/api/v1/musicon/recommended/albums", middleware.OptionalAuth(musicon.GetRecommendedAlbums))
// // 	router.GET("/api/v1/musicon/playlists/:playlistid/songs", middleware.OptionalAuth(musicon.GetPlaylistSongs))
// // }
