package cmd

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jhillyerd/go.enmime"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/fatih/set.v0"
)

type Options struct {
	Delimiter  string `short:"d" long:"delimiter" value-name:"DELIM" description:"use DELIM instead of COMMA for field delimiter." default:","`
	Digest     string `long:"digest" choice:"md5" choice:"sha1" description:"compute message digest for each email."`
	ListFields bool `short:"l" long:"list-fields" description:"list all metadata fields found."`
	Fields     []string `short:"f" long:"field" default:"all" value-name:"FIELD" description:"include FIELD in report."`
	Output     string `short:"o" long:"output" value-name:"FILE" description:"write output to FILE instead of stdout."`
	Version    bool `long:"version" description:"Show application version and exit."`
	Args struct {
		Files []string
	}   `positional-args:"yes" required:"yes" value-name:"FILE"`
}

func ListFields(opts *Options) error {
	paths, err := resolvePaths(opts.Args.Files)
	if err != nil {
		log.Fatal(err)
	}
	count := len(paths)
	_fields := set.New()

	bar := pb.StartNew(count)
	for i := 0; i < count; i++ {
		fh, err := os.Open(paths[i])
		if err != nil {
			log.Print(err)
			continue
		}

		reader := bufio.NewReader(fh)
		md, err := readMeta(reader)
		fh.Close()
		for k := range md {
			_fields.Add(k)
		}
		bar.Increment()
	}
	bar.Finish()

	fieldList := _fields.List()
	fields := make([]string, len(fieldList))
	for i, field := range fieldList {
		fields[i] = field.(string)
	}
	sort.Strings(fields)
	for _, field := range fields {
		fmt.Println(field)
	}

	return nil
}

func RunReport(opts *Options) error {
	paths, err := resolvePaths(opts.Args.Files)
	if err != nil {
		log.Fatal(err)
	}
	count := len(paths)
	report := []map[string]string{}
	fieldsHit := set.New()
	reqFields := set.New()
	for _, field := range opts.Fields {
		reqFields.Add(field)
	}
	m := md5.New()
	s := sha1.New()

	inclFilename := reqFields.Has("filename") || reqFields.Has("all")
	inclPath := reqFields.Has("path") || reqFields.Has("all")

	bar := pb.StartNew(count)
	for i := 0; i < count; i++ {
		fh, err := os.Open(paths[i])
		if err != nil {
			log.Println(err)
			continue
		}
		reader := bufio.NewReader(fh)
		buf := new(bytes.Buffer)
		buf.ReadFrom(reader)
		fh.Close()

		md, err := readMeta(bufio.NewReader(bytes.NewReader(buf.Bytes())))

		if opts.Digest == "md5" {
			m.Reset()
			_, err := buf.WriteTo(m)
			if err != nil {
				log.Println(err)
			}
			md["md5"] = hex.EncodeToString(m.Sum(nil))
		} else if opts.Digest == "sha1" {
			s.Reset()
			_, err := buf.WriteTo(s)
			if err != nil {
				log.Println(err)
			}
			md["sha1"] = hex.EncodeToString(s.Sum(nil))
		}
		if inclFilename {
			md["filename"] = filepath.Base(paths[i])
		}
		if inclPath {
			md["path"] = paths[i]
		}

		report = append(report, md)
		for k := range md {
			fieldsHit.Add(k)
		}
		bar.Increment()
	}
	bar.Finish()

	var fields []string
	if reqFields.Has("all") {
		fields = make([]string, fieldsHit.Size())
		fieldList := fieldsHit.List()
		for i, field := range fieldList {
			fields[i] = field.(string)
		}
		sort.Strings(fields)
	} else {
		fields = opts.Fields
	}
	if opts.Digest != "" {
		fields = append(fields, opts.Digest)
	}

	if opts.Output == "" {
		printReport(fields, report)
	} else {
		writeReport(fields, report, opts)
	}

	return nil
}

func resolvePaths(paths []string) ([]string, error) {
	absPaths := []string{}

	for _, path := range paths {
		gPaths, err := filepath.Glob(path)
		if err != nil {
			continue
		}

		for _, gPath := range gPaths {
			absPath, _ := filepath.Abs(gPath)
			mode, err := os.Stat(absPath)

			if err != nil {
				return nil, err
			}

			if mode.IsDir() {
				filepath.Walk(absPath, find(&absPaths))
			} else {
				absPaths = append(absPaths, absPath)
			}
		}
	}

	return absPaths, nil
}

func readMeta(reader *bufio.Reader) (map[string]string, error) {
	doc, err := enmime.ParseMIME(reader)
	if err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	for header, val := range doc.Header() {
		headers[strings.ToLower(header)] = enmime.DecodeHeader(val[0])
	}

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

func printReport(fields []string, report []map[string]string) {
	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetBorder(false)
	tw.SetColumnSeparator("")
	tw.SetHeaderLine(false)
	tw.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tw.SetHeader(fields)

	row := make([]string, len(fields))
	for i := 0; i < len(report); i++ {
		for j, field := range fields {
			row[j] = report[i][field]
		}
		tw.Append(row)
	}
	tw.Render()
}

func writeReport(fields []string, report []map[string]string, opts *Options) {
	d := opts.Delimiter

	f, err := os.Create(opts.Output)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	row := make([]string, len(fields))
	for i, field := range fields {
		row[i] = field
	}
	fmt.Fprintln(w, strings.Join(row, d))
	for i := 0; i < len(report); i++ {
		for j, field := range fields {
			row[j] = report[i][field]
		}
		fmt.Fprintln(w, strings.Join(row, d))
	}
	w.Flush()
}