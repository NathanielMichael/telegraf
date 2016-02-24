package nfs

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"bytes"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type NFS struct {
	NFSStatBin string
}

var sampleConfig = `
  # The path to nfsstat binary defaults to /usr/sbin/nfsstat
  NFSStatBin = "/usr/sbin/nfsstat"
`

func (n *NFS) SampleConfig() string {
	return sampleConfig
}

func (n *NFS) Description() string {
	return `Reads 'nfsstat' stats`
}

func (n *NFS) Gather(acc telegraf.Accumulator) error {
	if len(n.NFSStatBin) == 0 {
		fmt.Fprintln(os.Stderr, "Path to nfsstat bin required. skipping.")
		return nil
	}

	if _, err := os.Stat(n.NFSStatBin); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s is missing. skipping.", n.NFSStatBin)
		return nil
	}

	cmdArgs := []string{}

	cmd := exec.Command(n.NFSStatBin, cmdArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmd.StdoutPipe error: %s\n", err)
		return nil
	}

	if err = cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "cmd.Start error: %s\n", err)
		return nil
	}

	scanner := bufio.NewScanner(cmdReader)
	for scanner.Scan() {
		line := scanner.Text()
		re := regexp.MustCompile(`(\w+)\s+(\w+)\s+(\w+)\s+(\w+):\s+(\w+)`)
		parts := re.FindStringSubmatch(string(line))

		if len(parts) != 6 {
			continue
		}

		tags := map[string]string{"nfs_version": parts[1], "nfs_type": parts[2]}

		measurement := fmt.Sprintf("%s_%s_%s_%s", parts[1], parts[2], parts[3], parts[4])

		sValue := string(parts[5])

		iVal, err := strconv.ParseInt(sValue, 10, 64)
		if err == nil {
			acc.Add(measurement, iVal, tags)
		} else {
			acc.Add(measurement, sValue, tags)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scanner.Err error: %s\n", err)
	}

	if err = cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "cmd.Wait error: %s\n", err)
		fmt.Fprintln(os.Stderr, stderr.String())
		return nil
	}

	return nil
}

func init() {
	inputs.Add("nfs", func() telegraf.Input {
		return &NFS{}
	})
}
