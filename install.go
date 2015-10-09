package main

/* Install program for Gumshoe.
 * Once you have downloaded the gumshoe package, run install to put the gumshoe files in the
 * default locations. Running install -h will give you the list of flags to set your own
 * environment.
 */

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

var (
	installPath = flag.String("install_dir", "/usr/local/gumshoe", "Location for Gumshoe run files.")
	userPath    = flag.String("user_dir", filepath.Join(os.Getenv("HOME"), ".gumshoe"), "Location for user configuration and data.")
	cpuArch     = flag.String("arch", "", "CPU Architecture to compile gumshoe under. If unset, we will determine to the best of our ability.")
)

type CfgTemplate struct {
	InstallDir, UserDir, DataDir, DlDir, FetchDir, LogDir string
}

func main() {
	flag.Parse()
	whereAmI := filepath.Dir(os.Args[0])

	err := os.Mkdir(*installPath, 0555)
	if err != nil {
		log.Fatalf("Unable to create directory %s probably because you don't have permission.\n", *installPath)
	}
	log.Println("Copying gumshoe data to the install directory.")
  cpWWW := exec.Command("cp", "-r", filepath.Join(whereAmI, "www"), *installPath)
	err = cpWWW.Run()
	if err != nil {
		log.Fatalf("Failure to copy data files: %s\n", err)
	}

	log.Println("Creating user data directory.")
	for _, dir := range []string{"completed", "data", "fetch", "logs"} {
		err = os.MkdirAll(filepath.Join(*userPath, dir), 0777)
		if err != nil {
			log.Fatalf("Failure to setup user dir: %s\n", err)
		}
	}

	err = CopyConfig(whereAmI)
	if err != nil {
		log.Fatalln("Error editing config for the first time.")
	}
	SetCPUArch()

  // Set the go binary install path via GOBIN, install, then unset the variable.
	err = os.Setenv("GOBIN", *installPath)
	if err != nil {
		log.Fatalf("Can't set environment variables. %s\n", err)
	}
	installGumshoe := exec.Command("go", "install", "-i", filepath.Join(whereAmI, "gumshoed.go"))
	err = installGumshoe.Run()
	if err != nil {
		log.Fatalf("Error building gumshoe: %s\n", err)
	}
  err = os.Unsetenv("GOBIN")
  if err != nil {
    log.Fatalln(err)
  }

	log.Printf("Gumshoe installed successfully. Set $PATH to include %s\n", *installPath)
}

func SetCPUArch() {
	unameCmd := exec.Command("uname", "-m")
	out, _ := unameCmd.Output()
  sOut := string(out)
	switch {
	case sOut == "x86_64":
		os.Setenv("GOARCH", "amd64")
	case sOut == "ppc64":
		os.Setenv("GOARCH", "ppc64")
	default:
		os.Setenv("GOARCH", "x86")
	}
}

func CopyConfig(gct string) error {
	usrCfg, err := os.Create(filepath.Join(*installPath, "gumshoe.cfg"))
	if err != nil {
		return nil
	}
	ct := CfgTemplate{
		*installPath, *userPath, filepath.Join(*userPath, "data"), filepath.Join(*userPath, "completed"),
		filepath.Join(*userPath, "fetch"), filepath.Join(*userPath, "logs"),
	}
	t := template.Must(template.New("config").ParseFiles(filepath.Join(gct, "cfg", "gumshoe.cfg")))
	return t.Execute(usrCfg, ct)
}
