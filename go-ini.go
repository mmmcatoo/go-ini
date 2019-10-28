package go_ini

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
)

var lineBreaker string

// Initializes the newline character
func init() {
	if runtime.
		GOOS == "windows" {
		lineBreaker = "\r\n"
	} else {
		lineBreaker = "\n"
	}
}

type IniReader struct {
	stage map[string]string
}

func NewReader(content string) (*IniReader, error) {
	var iniReader IniReader
	// Start analyzing the text
	result := iniReader.formatText(content)
	iniReader.stage = result
	return &iniReader, nil
}

func NewByteReader(bytes []byte) (*IniReader, error) {
	return NewReader(string(bytes))
}

func NewFileReader(path string) (*IniReader, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() {
		return nil, errors.New("the expected input is not a folder but a file")
	}
	buffBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewByteReader(buffBytes)
}

func NewIoReader(reader io.Reader) (*IniReader, error) {
	buffBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return NewByteReader(buffBytes)
}

func NewResponseReader(response http.Response) (*IniReader, error) {
	return NewIoReader(response.Body)
}

func (this *IniReader) formatText(raw string) map[string]string {
	var (
		sectionText string
		key         string
		value       string
		prevKey     string
	)
	result := make(map[string]string)
	contentArray := strings.Split(raw, lineBreaker)
	for _, v := range contentArray {
		if len(strings.TrimSpace(v)) > 0 {
			firstChr := strings.TrimSpace(v[0:1])
			if firstChr == ";" || firstChr == "/" {
				// Skip comment content
				continue
			}
			if firstChr == "[" {
				// Get right brackets position
				endPosition := strings.Index(v, "]")
				sectionText = v[1:endPosition]
				continue
			}
			if len(firstChr) > 0 && firstChr != "\t" {
				equalPosition := strings.Index(v, "=")
				key = strings.TrimSpace(v[0:equalPosition])
				value = strings.TrimSpace(v[equalPosition+1:])
				prevKey = fmt.Sprintf("%s.%s", sectionText, key)
				result[prevKey] = value
			} else {
				// Multiple rows of data
				result[prevKey] = fmt.Sprintf("%s%s%s", result[prevKey], lineBreaker, v)
			}
		}
	}
	return result
}

func (this *IniReader) GetValue(field string) (string, error) {
	return this.GetByDot("", field)
}

func (this *IniReader) GetSectionValue(section, field string) (string, error) {
	return this.GetByDot(section, field)
}

func (this *IniReader) GetByDot(dotKey ...string) (string, error) {
	val, ok := this.stage[strings.Join(dotKey, ".")]
	if ok {
		// Replace placeholder like %(VARS)s into right value
		pattern := regexp.MustCompile("%\\(.*?\\)s")
		groups := pattern.FindAllString(val, -1)
		if len(groups) > 0 {
			for _, text := range groups {
				vars := strings.Replace(strings.Replace(text, ")s", "", -1), "%(", "", -1)
				replaceVars, err := this.GetByDot(vars)
				if err == nil {
					val = strings.Replace(val, text, replaceVars, 1)
				}
			}
		}
		return val, nil
	}
	return "", errors.New("the key name for the query does not exist")
}
