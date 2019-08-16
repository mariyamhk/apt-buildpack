package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"io/ioutil"

	"github.com/cloudfoundry/apt-buildpack/src/apt/apt"
	"github.com/cloudfoundry/apt-buildpack/src/apt/supply"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	logger := libbuildpack.NewLogger(os.Stdout)

	if os.Getenv("BP_DEBUG") != "" {
		cmd := exec.Command("find", ".")
		cmd.Dir = "/tmp/cache"
		cmd.Stdout = os.Stdout
		cmd.Run()
	}

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		logger.Error("Unable to determine buildpack directory: %s", err.Error())
		os.Exit(9)
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		logger.Error("Unable to load buildpack manifest: %s", err.Error())
		os.Exit(10)
	}

	stager := libbuildpack.NewStager(os.Args[1:], logger, manifest)
	if err := stager.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	if err = stager.SetStagingEnvironment(); err != nil {
		logger.Error("Unable to setup environment variables: %s", err.Error())
		os.Exit(13)
	}
		
	aptFileFromStager := filepath.Join(stager.BuildDir(), "apt.yml")
	logger.Info("Path to apt.yml: %s", aptFileFromStager)

	currentDir, dirErr := filepath.Abs(filepath.Dir(os.Args[0]))
	if dirErr != nil {
		logger.Error(dirErr.Error())
	}
	logger.Info("Path of executing file: %s", currentDir)

	osex, oserr := os.Executable()
	if err != nil {
		logger.Error("OS executable error: %s", oserr)
	}
	expath := filepath.Dir(osex)
	logger.Info("OS Executable path: %s", expath)

	b1, apterr1 := ioutil.ReadFile("apt.yml") // just pass the file name
	if apterr1 != nil {
		logger.Error("Error reading apt from apt.yml")
		logger.Error(apterr1.Error())
	}
	logger.Info("File contents from apt.yml: %s", string(b1))
	
	/*
	
	if exists, err := libbuildpack.FileExists(filepath.Join(stager.BuildDir(), "apt.yml")); err != nil {
		logger.Error("Unable to test existence of apt.yml: %s", err.Error())
		os.Exit(16)
	} else if !exists {
		logger.Error("Apt buildpack requires apt.yml\n(https://github.com/cloudfoundry/apt-buildpack/blob/master/fixtures/simple/apt.yml)")
		if exists, err := libbuildpack.FileExists(filepath.Join(stager.BuildDir(), "Aptfile")); err != nil || exists {
			logger.Error("Aptfile is deprecated. Please convert to apt.yml")
		}
		os.Exit(17)
	}
	*/
	
	//filepath.Join(stager.BuildDir(), "apt.yml")
	aptContent := `---
packages:
- cups
- cups-client`

	command := &libbuildpack.Command{}
	a := apt.New(command, aptContent, "/etc/apt", stager.CacheDir(), filepath.Join(stager.DepDir(), "apt"))
	if err := a.Setup(); err != nil {
		logger.Error("Unable to initialize apt package: %s", err.Error())
		os.Exit(13)
	}

	supplier := supply.New(stager, a, logger)

	if err := supplier.Run(); err != nil {
		logger.Error("Error running supply: %s", err.Error())
		os.Exit(14)
	}

	if err := stager.WriteConfigYml(nil); err != nil {
		logger.Error("Error writing config.yml: %s", err.Error())
		os.Exit(15)
	}

	stager.StagingComplete()
}
