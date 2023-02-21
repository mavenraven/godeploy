package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
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
	err := rootCmd.Execute()
	if err != nil {
		rootCmd.Usage()
		os.Exit(1)
	}
}

func init() {
	flags.root.port = rootCmd.PersistentFlags().IntP("port", "", 22, "port number of your server")
	flags.root.host = rootCmd.PersistentFlags().StringP("host", "", "", "host name or IP address of your server")
	flags.root.key = rootCmd.PersistentFlags().StringP("key", "", "", "location of id_rsa. Defaults to $HOME/.ssh/id_rsa")
	rootCmd.MarkPersistentFlagRequired("host")
}
