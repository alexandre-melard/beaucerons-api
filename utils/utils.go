package utils

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
)

type Data struct {
	Data string `json:"response"`
}

type TreeCleanUp struct {
	regex       *regexp.Regexp
	replacement string
}

func WriteResponse(response io.Reader, w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	io.Copy(w, response)
}

func ResponseJSON(data string, w http.ResponseWriter, statusCode int) {
	response := Data{data}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	Response(jsonResponse, w, statusCode)
}

func Response(json []byte, w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(json)
}

func CleanUpTreeResponse(tree []byte, w http.ResponseWriter) []byte {
	regexps := [10]TreeCleanUp{
		{regexp.MustCompile(`"`), ""},                                                        // 0
		{regexp.MustCompile(`}: {{`), ", \"children\": [{"},                                  // 1
		{regexp.MustCompile(`\[{{`), "{"},                                                    // 2
		{regexp.MustCompile(`}}\]`), "}"},                                                    // 3
		{regexp.MustCompile(`: {}`), ""},                                                     // 4
		{regexp.MustCompile(`}}`), "}]}"},                                                    // 5
		{regexp.MustCompile(`}}`), "}]}"},                                                    // 6
		{regexp.MustCompile(`=`), ":"},                                                       // 7
		{regexp.MustCompile(`name:`), "\"name\":\""},                                         // 8
		{regexp.MustCompile(`, uuid:([\w-]*)`), "\", \"uuid\":\"$1\", \"link\":\"/dog/$1\""}} // 9
	if len(tree) > 1 {
		tree = tree[1 : len(tree)-1]
	}
	for _, treeCleanUp := range regexps {
		tree = treeCleanUp.regex.ReplaceAll(tree, []byte(treeCleanUp.replacement))
	}
	return tree
}
