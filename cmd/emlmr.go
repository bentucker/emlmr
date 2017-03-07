package cmd

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhillyerd/go.enmime"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/fatih/set.v0"
)

func RunReport(files []string, fields []string) error {
	absPaths := []string{}
	for _, path := range files {
		absPath, _ := filepath.Abs(path)

		mode, err := os.Stat(absPath)

		if err != nil {
			return err
		}

		if mode.IsDir() {
			filepath.Walk(absPath, find(&absPaths))
		} else {
			absPaths[0] = absPath
		}
	}

	count := len(absPaths)
	report := []map[string]string{}
	_fields := set.New()

	bar := pb.StartNew(count)
	for i := 0; i < count; i++ {
		fh, err := os.Open(absPaths[i])
		if err != nil {
			log.Print(err)
			continue
		}

		md, err := readMeta(fh)
		report = append(report, md)
		for k := range md {
			_fields.Add(k)
		}
		bar.Increment()
	}
	bar.Finish()

	if len(fields) == 1 && fields[0] == "all" {
		fields = make([]string, _fields.Size())
		fieldList := _fields.List()
		for i, field := range fieldList {
			fields[i] = field.(string)
		}
	}

	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetBorder(false)
	tw.SetColumnSeparator("")
	tw.SetHeaderLine(false)
	tw.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tw.SetHeader(fields)

	row := make([]string, len(fields))
	for i := 0; i < count; i++ {
		for j, field := range fields {
			row[j] = report[i][field]
		}
		tw.Append(row)
	}
	tw.Render()

	return nil
}

func readMeta(fh *os.File) (map[string]string, error) {
	reader := bufio.NewReader(fh)

	doc, err := enmime.ParseMIME(reader)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	for header, val := range doc.Header() {
		headers[strings.ToLower(header)] = enmime.DecodeHeader(val[0])
	}

	fh.Close()
	return headers, nil
}

func find(paths *[]string) func(path string, info os.FileInfo,
	err error) error {

	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			*paths = append(*paths, path)
		}

		return nil
	}
}
