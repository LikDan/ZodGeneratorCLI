package handlers

import (
	"github.com/urfave/cli/v2"
	"zodGeneratorCLI/internal/controllers"
)

type Handler interface {
	Run(arguments []string) (err error)
}

type handler struct {
	*cli.App
	controller controllers.Controller
}

func NewHandler(controller controllers.Controller) Handler {
	h := &handler{
		controller: controller,
	}
	h.App = &cli.App{
		Commands: []*cli.Command{
			{
				Name:   "generate",
				Action: h.Generate,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "workingDir", Value: "."},
					&cli.BoolFlag{Name: "recursive"},
					&cli.StringSliceFlag{Name: "files", Aliases: []string{"file"}},
				},
			},
		},
	}

	return h
}

func (h *handler) Generate(ctx *cli.Context) error {
	workingDir := ctx.String("workingDir")
	recursive := ctx.Bool("recursive")
	files := ctx.StringSlice("files")

	return h.controller.Generate(ctx.Context, workingDir, recursive, files)
}
