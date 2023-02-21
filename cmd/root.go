package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "snakeplant",
	Short: "snakeplant provides a way to easily deploy to a single machine",
	Long: `snakeplant 
examples and usage of using your application. For example:

Much like a real snake plant, the idea is that 'snakeplant' will make your
server a low maintenance and enjoyable experience. And also like a real
snake plant, you will likely have to OCASIONALLY actual ssh into your server
and do tasks. 

So, the goal isn't to completely hide sysadmin work from you. That never works.
The goal is to provide you with the education that you need to fix stuf when it goes wrong.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntP("port", "", 22, "port number of your server")
	rootCmd.Flags().StringP("host", "", "", "host name or IP address of your server")
}
