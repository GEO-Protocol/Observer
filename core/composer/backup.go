package composer

import (
	"fmt"
	"geo-observers-blockchain/core/chain/chain"
	"geo-observers-blockchain/core/common/errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	// tempFilesExtension represents extension for all temporary copies of chain data.
	tempFilesExtension = ".temp"
)

func (c *Composer) createOrReplaceChainBackup() (tmpCopyFilename string, e errors.E) {
	c.dropChainBackupIfAny()

	tmpCopyFilename = fmt.Sprint(chain.DataFilePath, "_", time.Now().Unix(), tempFilesExtension)

	cmd := exec.Command("cp", "--reflink", chain.DataFilePath, tmpCopyFilename)
	err := cmd.Run()

	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return
}

func (c *Composer) restoreChainBackup(backupFilePath string) (e errors.E) {
	// Check if file exists.
	if _, err := os.Stat(backupFilePath); os.IsNotExist(err) {
		e = errors.AppendStackTrace(err)
		return
	}

	// Attempt to remove current chain data.
	// WARN: commands order is necessary!
	cmd := exec.Command("rm", chain.DataFilePath)
	err := cmd.Run()
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	cmd = exec.Command("mv", backupFilePath, chain.DataFilePath)
	err = cmd.Run()
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return
}

// dropChainBackupIfAny removes all files with "tempFilesExtension" that are located in data dir.
// The case when there are no such files is not considered as an error (the call would silently finish).
func (c *Composer) dropChainBackupIfAny() (e errors.E) {
	files, err := filepath.Glob(chain.DataPath + "*" + tempFilesExtension)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	for _, f := range files {
		if err := os.Remove(f); err != nil {
			e = errors.AppendStackTrace(err)
			return
		}
	}

	return
}
