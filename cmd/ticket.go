package cmd

import "github.com/infinitybotlist/iblfile"

func init() {
	iblfile.RegisterFormat(
		"ticket",
		&iblfile.Format{
			Format:  "transcript",
			Version: "a1",
		},
	)
}
