module github.com/usecakework/cakework/cli

go 1.19

replace github.com/usecakework/cakework/lib => ../lib

require (
	github.com/sirupsen/logrus v1.9.0
	github.com/urfave/cli/v2 v2.23.7
	github.com/usecakework/cakework/lib v0.0.0-00010101000000-000000000000
)

require (
	github.com/MicahParks/keyfunc v1.9.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
)
