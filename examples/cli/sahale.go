package main

import (
    "fmt"
    "log"
    "os"

    "github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Commands: []*cli.Command{
            {
                Name:    "build",
                Aliases: []string{"a"},
                Usage:   "build your app in the sahale cloud",
                Action: func(cCtx *cli.Context) error {
                    fmt.Println("built app: ", cCtx.Args().First())
                    return nil
                },
            },
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
