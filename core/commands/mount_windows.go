package commands

import (
	"errors"

	cmds "gitlab.com/casperDev/Casper-server/commands"
)

var MountCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Not yet implemented on Windows.",
		ShortDescription: "Not yet implemented on Windows. :(",
	},

	Run: func(req cmds.Request, res cmds.Response) {
		res.SetError(errors.New("Mount isn't compatible with Windows yet"), cmds.ErrNormal)
	},
}
