package routes

import (
	"naevis/ratelim"

	"github.com/julienschmidt/httprouter"
)

func RoutesWrapper(router *httprouter.Router, rateLimiter *ratelim.RateLimiter) {
	AddMusicRoutes(router, rateLimiter)
}
