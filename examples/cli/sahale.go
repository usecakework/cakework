package main

import (
    "fmt"
    "log"
    "os"

    "github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name:  "sahale",
        Usage: "let's build a workflow! 🏔",
        Action: func(*cli.Context) error {
            fmt.Println("hello world!")
            return nil
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
