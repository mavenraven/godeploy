package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "snakeplant",
	Short: "'snakeplant' deploys to and maintains your server",
	Long: `snakeplant 
Much like a real snake plant, the idea is that 'snakeplant' will make caring for your
server a low maintenance and enjoyable experience. And also like a real
snake plant, you will likely have to OCCASIONALLY actually ssh into your server
and do tasks. 

So, the goal isn't to completely hide sysadmin work from you. That never works.
The goal is to provide you with the education that you need to fix stuff when it goes wrong.`,
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		RootCmd.Usage()
		os.Exit(1)
	}
}

func init() {
}
