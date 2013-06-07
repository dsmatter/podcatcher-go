package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"github.com/smatter0ne/podcatcher/rss"
)

type NewFiles struct {
	dir  string
	urls []string
}

func torrentEqual(a, b string) bool {
	return a == b || a+".torrent" == b || a == b+".torrent"
}

func newFiles(remoteFiles, localFiles []string) []string {
	// Optimization for the common case: we already have the most recent episode
	if len(remoteFiles) > 0 {
		localLookAhead := 5
		newestFn := path.Base(remoteFiles[0])

		// Assumption: file name order represents episode order
		// Scan list from end to beginning
		for i := len(localFiles) - 1; i >= 0 && i >= len(localFiles) - localLookAhead; i-- {
			if torrentEqual(newestFn, localFiles[i]) {
				return nil
			}
		}
	}

	// Create a set of local file names
	localFilesSet := make(map[string]bool)

	// Fill the set
	for _, fn := range localFiles {
		localFilesSet[fn] = true
		localFilesSet[fn+".torrent"] = true
	}

	for i, url := range remoteFiles {
		urlFn := path.Base(url)

		if localFilesSet[urlFn] {
			return remoteFiles[:i]
		}
	}
	return remoteFiles[:]
}

func checkDir(dir string, c chan NewFiles) {
	// Open the dir
	dirp, err := os.Open(dir)
	if err != nil {
		c <- NewFiles {dir, nil}
		return
	}

	cLocalFiles := make(chan []string)
	cRemoteFiles := make(chan []string)

	// Read all local files
	go func() {
		localFiles, _ := dirp.Readdirnames(0)
		cLocalFiles <- localFiles
	}()

	// Read the remote files
	go func() {
		remoteFiles, err := rss.FeedLinks(dir)
		if err != nil {
			fmt.Println("Error with feed " + dir + ": " + err.Error())
		}
		cRemoteFiles <- remoteFiles
	}()

	localFiles := <-cLocalFiles
	remoteFiles := <-cRemoteFiles
	newFiles := newFiles(remoteFiles, localFiles)
	c <- NewFiles{dir, newFiles}
}

func isResultEmpty(nfa []NewFiles) bool {
	for _, nf := range nfa {
		if len(nf.urls) > 0 {
			return false
		}
	}
	return true
}

func printNewFiles(nfa []NewFiles) {
	for _, nf := range nfa {
		if (len(nf.urls) < 1) {
			continue
		}
		fmt.Printf("> New files for %s:\n", nf.dir)
		for _, url := range nf.urls {
			fmt.Println(path.Base(url))
		}
	}
}

func downloadNewFiles(nfa []NewFiles) {
	// Look up the aria2c executable
	ariaPath, err := exec.LookPath("aria2c")
	if err != nil {
		fmt.Println("Please install the aria downloader")
		os.Exit(1)
	}

	for _, nf := range nfa {
		for _, url := range nf.urls {
			// Redirect STDIN, STDOUT and STDERR
			pAttr := os.ProcAttr {"",  // no chdir
														nil, // no custom env
														[]*os.File{os.Stdin, os.Stdout, os.Stderr},
														nil} // no process creation attrs
			pArgv := []string {ariaPath,
												 "--file-allocation=none",
												 "--seed-time=0",
												 "-d", nf.dir,
												 "-o", path.Base(url),
												 url}

			p, err := os.StartProcess(pArgv[0], pArgv, &pAttr)
			if err != nil {
				fmt.Printf("Error downloading %s:\n", path.Base(url))
				fmt.Println(err)
				continue
			}
			defer p.Wait()
		}
	}
}

func main() {
	filter := ""
	cNewFiles := make(chan NewFiles)
	numDirs := 0

	if (len(os.Args) > 1) {
		filter = os.Args[1]
	}

	currentDir, err := os.Open(".")
	if err != nil {
		fmt.Println("Error opening current directory")
		return
	}

	files, err := currentDir.Readdir(0)
	if err != nil {
		fmt.Println("Error reading local files")
		return
	}

	for _, info := range files {
		fn := info.Name()
		// Some checks
		if !info.IsDir() || fn == "." || fn == ".." || !strings.Contains(fn, filter) {
			continue
		}

		// Check for feed.url
		_, err = os.Stat(path.Join(fn, "feed.url"))
		if err != nil {
			continue
		}

		go checkDir(fn, cNewFiles)
		numDirs++
	}

	// Gather the results
	result := make([]NewFiles, numDirs)
	for i := 0; i < numDirs; i++ {
		result[i] = <-cNewFiles
		fmt.Println("Checked", result[i].dir)
	}

	// Quit if there is nothing to do
	if isResultEmpty(result) {
		return
	}

	// Show the results
	printNewFiles(result)

	// Ask to download
	fmt.Print("Download? (Y/n) > ")
	var ans string
	fmt.Scanln(&ans)

	// Act accordingly
	if len(ans) < 1 || ans[0] == 'y' || ans[0] == 'Y' || ans[0] == '\n' {
		downloadNewFiles(result)
	} else {
		fmt.Println("kthxbye")
	}
}
