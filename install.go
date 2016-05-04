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
  help        = flag.Bool("h", false, "Show this help documentation.")
  installSBin = flag.Bool("install_sbin", false, "Install the gumshoe binary in /usr/local/sbin?")
)

type CfgTemplate struct {
	InstallDir, UserDir, DataDir, DlDir, FetchDir, LogDir string
}

func SetCPUArch() error {
	unameCmd := exec.Command("uname", "-m")
	out, _ := unameCmd.Output()
  sOut := string(out)
	switch {
	case sOut == "x86_64":
		return os.Setenv("GOARCH", "amd64")
	case sOut == "ppc64":
		return os.Setenv("GOARCH", "ppc64")
	default:
		return os.Setenv("GOARCH", "x86")
	}
}

func CopyConfig(gct string) error {
	usrCfg, err := os.Create(filepath.Join(*userPath, "gumshoe.cfg"))
	if err != nil {
    log.Println("Unable to create a system-wide gunshoe config file. Continuing.")
		return nil
	}

	ct := CfgTemplate{
		*installPath,
    *userPath,
    filepath.Join(*userPath, "data"),
    filepath.Join(*userPath, "completed"),
		filepath.Join(*userPath, "fetch"),
    filepath.Join(*userPath, "logs"),
	}
	t := template.Must(template.New("gumshoe.cfg").ParseFiles(filepath.Join(gct, "cfg", "gumshoe.cfg")))
	return t.Execute(usrCfg, ct)
}

func main() {
	flag.Parse()
  if *help {
    flag.PrintDefaults()
    return
  }
  log.Println("Making necessary directories.")
	err := os.MkdirAll(filepath.Join(*installPath, "www"), os.ModeDir|0777)
  /*
	if err != nil {
    log.Println(err)
		log.Fatalf("Unable to create directory %s probably because you don't have permissions.\n", *installPath)
	}
	log.Println("Copying gumshoe data to the install directory.")
  cpWWW := exec.Command("/bin/cp", "-rf", filepath.Join(os.Getenv("PWD"), "www", "*"), filepath.Join(*installPath, "www"))
	err = cpWWW.Run()
	if err != nil {
    log.Println(err)
		log.Fatalf("Failure to copy data files: %s\n", err)
	}
*/
	log.Println("Creating user data directory.")
	for _, dir := range []string{"completed", "data", "fetch", "logs"} {
		err = os.MkdirAll(filepath.Join(*userPath, dir), 0777)
		if err != nil {
			log.Fatalf("Failure to setup user dir: %s\n", err)
		}
	}

	err = CopyConfig(os.Getenv("PWD"))
	if err != nil {
    log.Fatalf("Error creating user config file: %s\n", err)
	}

	err = SetCPUArch()
  if err != nil {
    log.Printf("E: %s\n", err)
    log.Println("Unable to determine system, going to try and build anyways.")
  }

  binpath := ""
  if *installSBin {
    binpath = "/usr/local/sbin"
  } else {
    binpath = filepath.Join(*installPath, "bin")
  }
	buildGumshoe := exec.Command("go", "build", "-o", filepath.Join(binpath, "gumshoe"), "main.go")
	err = buildGumshoe.Run()
	if err != nil {
		log.Fatalf("Error building gumshoe: %s\n", err)
	}

  // Flag to do this automatically
	log.Printf("Gumshoe installed successfully. Set $PATH to include %s\n", binpath)
}

