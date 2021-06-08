package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/alexandre-melard/beaucerons/api/auth"
	"github.com/alexandre-melard/beaucerons/api/utils"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	"github.com/form3tech-oss/jwt-go"
	"github.com/joho/godotenv"
)

type GremlinResult struct {
	Result []json.RawMessage `json:"result"`
}

func post(request string, script string) ([]byte, error) {
	client := &http.Client{}
	fmt.Println(request)
	req, err := http.NewRequest("POST", "http://host.docker.internal:2480/command/beaucerons/"+script, strings.NewReader(request))
	if err != nil {
		return nil, fmt.Errorf("the HTTP POST request creation failed with error %s", err)
	}
	req.Header.Add("Authorization", "Basic cm9vdDo3WVl4V1Rrc2NWcEFPTQ==")
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("the HTTP POST request execution failed with error %s", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the HTTP POST request answered with an error %v", response)
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading HTTP POST answser %v", err)
	}
	var gremlinResult GremlinResult
	err = json.Unmarshal(bodyBytes, &gremlinResult)
	if err != nil {
		return nil, fmt.Errorf("error reading HTTP POST answser %v", err)
	}
	if gremlinResult.Result == nil || len(gremlinResult.Result) == 0 {
		return nil, nil
	}
	var result string
	if len(gremlinResult.Result) > 1 {
		stringArray := []string{}
		for _, result := range gremlinResult.Result {
			stringArray = append(stringArray, string(result))
		}
		result = "[" + strings.Join(stringArray[:], ",") + "]"
	} else if len(gremlinResult.Result) == 1 {
		result = string(gremlinResult.Result[0])
	} else {
		return nil, nil
	}

	return []byte(result), nil
}

func getDog(w http.ResponseWriter, r *http.Request) {
	if auth.CheckPermission(w, r, "read:dog") != nil {
		return
	}
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	var request = fmt.Sprintf(
		"g.V().has('uuid', '%s')."+
			"project('id', 'type', 'name', 'uuid', 'ship', 'tattoo', 'cotation', 'dob', 'color', 'other')."+
			"  by(id).by(label).by('name').by('uuid').by('ship').by('tattoo').by('cotation').by('dob').by('color').by('other')", uuid)
	response, err := post(request, "gremlin")
	if err != nil {
		fmt.Printf("error in post request %v\n", err)
		w.WriteHeader(500)
	}
	utils.Response(response, w, http.StatusOK)
}

func getDogParents(w http.ResponseWriter, r *http.Request) {
	if auth.CheckPermission(w, r, "read:dog:parents") != nil {
		return
	}
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	var request = fmt.Sprintf(
		"g.V().has('uuid', '%s').in('Parent')."+
			"project('id', 'type', 'name', 'uuid', 'ship', 'tattoo', 'cotation', 'dob', 'color', 'other')"+
			"  .by(id).by(label).by('name').by('uuid').by('ship').by('tattoo').by('cotation').by('dob').by('color').by('other')", uuid)
	response, err := post(request, "gremlin")
	if err != nil {
		fmt.Printf("error in post request %v\n", err)
		w.WriteHeader(500)
	}
	utils.Response(response, w, http.StatusOK)
}

func getDogPedigree(w http.ResponseWriter, r *http.Request) {
	if auth.CheckPermission(w, r, "read:dog:pedigree") != nil {
		return
	}
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	depth := vars["depth"]

	var request = fmt.Sprintf(
		"g.V().has('uuid', '%s')."+
			"repeat("+
			"  timeLimit(3000)."+
			"  in('Parent')"+
			")."+
			"times(%s)."+
			"tree()."+
			"by(project('name', 'uuid')."+
			"  by('name')."+
			"  by('uuid'))", uuid, depth)
	response, err := post(request, "gremlin")
	if err != nil {
		fmt.Printf("error in post request %v\n", err)
		w.WriteHeader(500)
	}
	tree := utils.CleanUpTreeResponse(response, w)
	utils.Response(tree, w, http.StatusOK)
}

func getDogOffsprings(w http.ResponseWriter, r *http.Request) {
	if auth.CheckPermission(w, r, "read:dog:offsprings") != nil {
		return
	}
	vars := mux.Vars(r)
	uuid := vars["uuid"]
	depth := vars["depth"]
	var request = fmt.Sprintf(
		"g.V().has('uuid', '%s')."+
			"repeat("+
			"  timeLimit(3000)."+
			"  out('Parent')"+
			")."+
			"until("+
			"  outE().count()."+
			"is(0).or().is(%s))."+
			"tree()."+
			"by(project('name', 'uuid')."+
			"  by('name')."+
			"  by('uuid'))", uuid, depth)
	response, err := post(request, "gremlin")
	if err != nil {
		fmt.Printf("error in post request %v\n", err)
		w.WriteHeader(500)
	}
	tree := utils.CleanUpTreeResponse(response, w)
	utils.Response(tree, w, http.StatusOK)
}

func search(w http.ResponseWriter, r *http.Request) {
	if auth.CheckPermission(w, r, "search") != nil {
		return
	}
	vars := mux.Vars(r)
	term := vars["term"]
	limit, err := strconv.Atoi(vars["limit"])
	if err != nil {
		limit = 20
	}
	var request = fmt.Sprintf(
		"SELECT uuid, name, @CLASS as type FROM Named  WHERE SEARCH_CLASS('%s*') = true limit %d", term, limit)
	response, err := post(request, "sql")
	if err != nil {
		fmt.Printf("error in post request %v\n", err)
		w.WriteHeader(500)
	}
	utils.Response(response, w, http.StatusOK)
}

func handleRequests() {
	jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{
		ValidationKeyGetter: auth.CheckKey,
		SigningMethod:       jwt.SigningMethodRS256,
	})

	r := mux.NewRouter().StrictSlash(true)
	r.Handle("/api/search/{limit}/{term}", handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, http.HandlerFunc(search))))
	r.Handle("/api/dog/{uuid}", jwtMiddleware.Handler(handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, http.HandlerFunc(getDog)))))
	r.Handle("/api/dog/{uuid}/parents", handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, http.HandlerFunc(getDogParents))))
	r.Handle("/api/dog/{uuid}/pedigree/{depth}", handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, http.HandlerFunc(getDogPedigree))))
	r.Handle("/api/dog/{uuid}/offsprings/{depth}", handlers.CompressHandler(handlers.LoggingHandler(os.Stdout, http.HandlerFunc(getDogOffsprings))))

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":10000", nil))
}

func main() {
	fmt.Println("Rest API v2.11 - Mux Routers")
	err := godotenv.Load()
	if err != nil {
		log.Print("Error loading .env file")
	}

	handleRequests()
}
