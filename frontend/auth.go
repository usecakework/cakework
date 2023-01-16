package main

import (
	"fmt"
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
		apiKey := c.Request.Header.Get("X-Api-Key");
		
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
			if user != nil {
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
		fmt.Println("got scope in jwt token middleware: " + scope)

		claims, ok := c.Request.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
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
			fmt.Println("customClaims.Scope")
			fmt.Println(customClaims.Scope)
			c.IndentedJSON(http.StatusForbidden, `{"message":"Insufficient scope."}`)
			return
		}
	
		c.Next()
	}
}

// func hasRightScope(c *gin.Context, scope string) bool {
// 	claims, ok := c.Request.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
// 	if !ok {
// 		c.AbortWithStatusJSON(
// 			http.StatusInternalServerError,
// 			map[string]string{"message": "Failed to get validated JWT claims."},
// 		)
// 		return false
// 	}

// 	customClaims, ok := claims.CustomClaims.(*CustomClaimsExample)
// 	if !ok {
// 		c.AbortWithStatusJSON(
// 			http.StatusInternalServerError,
// 			map[string]string{"message": "Failed to cast custom JWT claims to specific type."},
// 		)
// 		return false
// 	}

// 	if len(customClaims.Scope) == 0 {
// 		c.AbortWithStatusJSON(
// 			http.StatusBadRequest,
// 			map[string]string{"message": "Scope in JWT claims was empty."},
// 		)
// 		return false
// 	}

// 	if !strings.Contains(customClaims.Scope, scope) {
// 		c.IndentedJSON(http.StatusForbidden, `{"message":"Insufficient scope."}`)
// 		return false
// 	}

// 	return true
// }