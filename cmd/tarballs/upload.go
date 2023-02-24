package tarballs

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/mavenraven/snakeplant/cmd"
	"github.com/sfreiberg/simplessh"
	"github.com/spf13/cobra"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Puts the contents of the current directory into a tarball and uploads it to your server.",
	Long:  `So cool`,
	Run:   upload,
}

func init() {
	rootTarballsCmd.AddCommand(uploadCmd)
	cmd.Flags.Root.Port = uploadCmd.Flags().IntP("port", "", 22, "The port number of the ssh daemon running on your server.")
	cmd.Flags.Root.Host = uploadCmd.Flags().StringP("host", "", "", "The host name or IP address of your server.")
	cmd.Flags.Root.Key = uploadCmd.Flags().StringP("key", "", "", "The location of your 'id_rsa' file. Defaults to $HOME/.ssh/id_rsa.")
	uploadCmd.MarkFlagRequired("host")
}

func upload(command *cobra.Command, args []string) {
	tarballName := createTarball()
	fmt.Println(tarballName)

	socket := fmt.Sprintf("%v:%v", *cmd.Flags.Root.Host, *cmd.Flags.Root.Port)

	client, err := simplessh.ConnectWithKeyFileTimeout(socket, "Root", *cmd.Flags.Root.Key, 5*time.Second)
	cmd.AssertNoErr(err, "Unable to establish a connection.")

	remoteTempFileNameBytes, err := client.Exec("mktemp")
	cmd.AssertNoErr(err, "Unable to create temp file on server.")

	remoteTempFileName := strings.TrimSpace(string(remoteTempFileNameBytes))

	fmt.Printf("uploading tarball to %v at %v...\n", remoteTempFileName, time.Now().Format("15:04:05"))

	//client.Upload is unusably slow, just shell out instead for now.
	scpArgs := make([]string, 0)
	if *cmd.Flags.Root.Key != "" {
		args = append(args, "-i", *cmd.Flags.Root.Key)
	}
	scpArgs = append(args, tarballName, fmt.Sprintf("%v@%v:%v", "root", *cmd.Flags.Root.Host, remoteTempFileName))

	scpCmd := exec.Command("scp", scpArgs...)
	output, err := scpCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%v: %v\n", string(output), err)
		os.Exit(1)
	}
	fmt.Printf("tarball uploaded at %v\n", time.Now().Format("15:04:05"))

}

func createTarball() string {
	tarballFile, err := os.CreateTemp("", "")
	cmd.AssertNoErr(err, "unable to create tarball file")

	fmt.Printf("creating tarball of files to upload: %v...\n", tarballFile.Name())
	defer tarballFile.Close()

	gzipWriter := gzip.NewWriter(tarballFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	wd, err := os.Getwd()
	cmd.AssertNoErr(err, "Could not get current working directory to walk tarball tree.")

	filepath.Walk(wd, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if strings.Contains(path, ".git") {
			//TODO: we could use the gitignore to filter out other junk
			return nil
		}

		cmd.AssertNoErr(err, fmt.Sprintf("Could not walk into %v.", path))

		fileToAdd, err := os.Open(path)
		cmd.AssertNoErr(err, fmt.Sprintf("Could not open '%v' to add to tarball.", path))
		defer fileToAdd.Close()

		stat, err := fileToAdd.Stat()
		cmd.AssertNoErr(err, fmt.Sprintf("Could not get stat of '%v' to add to tarball", path))

		header := &tar.Header{
			Name:    path[len(wd)+1:],
			Size:    stat.Size(),
			Mode:    int64(stat.Mode()),
			ModTime: stat.ModTime(),
		}

		err = tarWriter.WriteHeader(header)
		cmd.AssertNoErr(err, fmt.Sprintf("Could not write header for '%v' in tarball", path))

		_, err = io.Copy(tarWriter, fileToAdd)
		cmd.AssertNoErr(err, fmt.Sprintf("Could not copy '%v' into tarball", path))
		return nil
	})

	return tarballFile.Name()
}
