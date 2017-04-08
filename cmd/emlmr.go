package cmd

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	set "github.com/emirpasic/gods/sets/hashset"
	stack "github.com/emirpasic/gods/stacks/arraystack"
	"github.com/jhillyerd/go.enmime"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/cheggaaa/pb.v1"
)

type Options struct {
	Delimiter  string   `short:"d" long:"delimiter" value-name:"DELIM" description:"use DELIM instead of COMMA for field delimiter" default:","`
	Digest     []string `long:"digest" choice:"md5" choice:"sha1" description:"compute message digest for each email"`
	Fields     []string `short:"f" long:"field" default:"all" value-name:"FIELD" description:"include FIELD in report"`
	ListFields bool     `short:"l" long:"list-fields" description:"list all metadata fields found"`
	Output     string   `short:"o" long:"output" value-name:"FILE" description:"write output to FILE instead of stdout"`
	Recursive  bool     `short:"r" long:"recursive" description:"read all EML files under each directory, recursively"`
	Version    bool     `long:"version" description:"Show application version and exit"`
	Args struct {
		Files []string `positional-arg-name:"FILE"`
	} `positional-args:"yes" required:"yes"`
}

func ListFields(opts *Options) {
	paths := resolvePaths(opts.Args.Files, opts.Recursive)
	count := len(paths)
	_fields := set.New()

	bar := pb.StartNew(count)
	for i := 0; i < count; i++ {
		fh, err := os.Open(paths[i])
		if err != nil {
			log.Println(err)
			continue
		}

		reader := bufio.NewReader(fh)
		md, err := readMeta(reader)
		fh.Close()
		if err != nil {
			log.Printf("Error parsing %s: %s", paths[i], err.Error())
			continue
		}
		for k := range md {
			_fields.Add(k)
		}
		bar.Increment()
	}
	bar.Finish()

	fieldList := _fields.Values()
	fields := make([]string, len(fieldList))
	for i, field := range fieldList {
		fields[i] = field.(string)
	}
	sort.Strings(fields)
	for _, field := range fields {
		fmt.Println(field)
	}
}

func RunReport(opts *Options) {
	paths := resolvePaths(opts.Args.Files, opts.Recursive)
	count := len(paths)
	report := []map[string]string{}
	fieldsHit := set.New()
	reqFields := set.New()
	for _, field := range opts.Fields {
		reqFields.Add(field)
	}
	m := md5.New()
	s := sha1.New()

	digests := set.New()
	for _, hash := range opts.Digest {
		digests.Add(hash)
	}

	inclFilename := reqFields.Contains("filename") || reqFields.Contains("all")
	inclPath := reqFields.Contains("path") || reqFields.Contains("all")

	bar := pb.StartNew(count)
	for i := 0; i < count; i++ {
		fh, err := os.Open(paths[i])
		if err != nil {
			log.Println(err)
			continue
		}

		// read the file contents
		reader := bufio.NewReader(fh)
		buf := new(bytes.Buffer)
		buf.ReadFrom(reader)
		fh.Close()

		md, err := readMeta(bufio.NewReader(bytes.NewReader(buf.Bytes())))
		if err != nil {
			log.Printf("Error parsing %s: %s", paths[i], err.Error())
			continue
		}

		// calculate message digest
		if digests.Contains("md5") {
			m.Reset()
			_, err := m.Write(buf.Bytes())
			if err != nil {
				log.Println(err)
			}
			md["md5"] = hex.EncodeToString(m.Sum(nil))
		}
		if digests.Contains("sha1") {
			s.Reset()
			_, err := s.Write(buf.Bytes())
			if err != nil {
				log.Println(err)
			}
			md["sha1"] = hex.EncodeToString(s.Sum(nil))
		}

		// add filename and path fields
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

	// decide which fields to include in the report
	var fields []string
	if reqFields.Contains("all") {
		// if "all" fields were requested, use all the fields that were hit
		// during processing
		fields = make([]string, fieldsHit.Size())
		fieldList := fieldsHit.Values()
		for i, field := range fieldList {
			fields[i] = field.(string)
		}
		sort.Strings(fields)
	} else {
		// otherwise, just use the fields specified on the command line
		fields = opts.Fields
	}

	// include digest field in report
	if !digests.Empty() {
		fields = append(fields, opts.Digest...)
	}

	if opts.Output == "" {
		printReport(fields, report)
	} else {
		writeReport(fields, report, opts)
	}
}

func resolvePaths(paths []string, recursive bool) []string {
	absPaths := []string{}

	// push all of the initial paths onto a stack
	pathStack := stack.New()
	for _, path := range paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			matches = []string{path}
		}

		for _, path2 := range matches {
			absPath, err := filepath.Abs(path2)
			if err != nil {
				log.Println(err)
				continue
			}
			pathStack.Push(absPath)
		}
	}

	// add all the files in each path in the pathStack to absPaths
	for empty := pathStack.Empty(); empty == false; {
		path, ok := pathStack.Pop()
		if !ok {
			break
		}
		absPath, err := filepath.Abs(path.(string))
		if err != nil {
			log.Println(err)
			continue
		}
		mode, err := os.Stat(absPath)

		if err != nil {
			log.Println(err)
			continue
		}

		if mode.IsDir() && recursive {
			// if recursion is enabled and we hit a directory, add all the
			// paths in that subdirectory to the pathStack so that we can
			// find the files contained in each subdirectory
			children, err := ioutil.ReadDir(absPath)
			if err != nil {
				log.Println(err)
				continue
			}
			for _, child := range children {
				pathStack.Push(filepath.Join(absPath, child.Name()))
			}
		} else if !mode.IsDir() {
			// add each file to the absPaths list
			absPaths = append(absPaths, absPath)
		}
	}

	return absPaths
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
