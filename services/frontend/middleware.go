package main

import (
	"net/http"
	"strings"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func abortWithStatusJSON(c *gin.Context, code int, message interface{}) {
	c.AbortWithStatusJSON(code, gin.H{"error": message})
}

func apiKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.Request.Header.Get("X-Api-Key")

		if apiKey == "" {
			abortWithStatusJSON(c, 401, "API token required")
			return
		} else {
			// check that api key is valid
			user, err := getUserFromAPIKey(apiKey)
			if err != nil {
				log.Error(err) // Q: how should we expose this error to the user?
				abortWithStatusJSON(c, 500, "Error while fetching user using API key")
				return
			}
			if user != nil && user.Id != "" {
				c.Request.Header.Add("userId", user.Id)
				c.Next()
			} else {
				abortWithStatusJSON(c, 401, "Invalid API token")
				return
			}
		}
	}
}

// this validates the the functions have the correct scopes
func jwtTokenMiddleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {

		claims, ok := c.Request.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
		if scope == "external" {
			sub := claims.RegisteredClaims.Subject
			log.Debug("registered claims")
			log.Debug(claims.RegisteredClaims)
			if sub == "" {
				abortWithStatusJSON(c, 401, "Invalid JWT token; missing sub claim; cannot find user id")
				return
			}
			userId := strings.Split(claims.RegisteredClaims.Subject, "|")[1]
			c.Request.Header.Add("userId", userId)
		}	

		if !ok {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				map[string]string{"message": "Failed to get validated JWT claims."},
			)
			return
		}

		customClaims, ok := claims.CustomClaims.(*CustomClaimsExample)
		if !ok {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				map[string]string{"message": "Failed to cast custom JWT claims to specific type."},
			)
			return
		}

		if len(customClaims.Scope) == 0 {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				map[string]string{"message": "Scope in JWT claims was empty."},
			)
			return
		}

		if scope == "" {
			log.Error("Middleware to inject scope into gin context failed; not able to finish authorizing JWT token")
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				map[string]string{"message": "sorry :( something broke, come talk to us"},
			)
			return
		}

		if !strings.Contains(customClaims.Scope, scope) {
			// get the scope from the token
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				map[string]string{"message": "Insufficient scope"},
			)
			return
		}

		c.Next()
	}
}
