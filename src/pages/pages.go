package pages

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/eknkc/amber"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"html/template"
	"io/ioutil"
	"net/http"
	"playerElo"
)

const adSpreadsheetFmtStr string = "http://docs.google.com/spreadsheets/d/e/2PACX-1vShbdRxaYapPkcDxBqGJexfp0cVZHVDQ3ZZtMukOaubBcLUnYrqC8ZetZZqOj1W7ln-XyzBu_6XB8Zv/pub?output=csv&gid=%s"

type PostgresConf struct {
	Username string
	Password string
	Host string
	Database string
}

type TemplateConf struct {
	TemplateDirectory string
}

type BasePathsConf struct {
	EloBasePath string
}

type Conf struct {
	PostgresConf
	TemplateConf
	BasePathsConf
}

type Spreadsheet struct {
	Headers []string
	Records [][]string
}

type Member struct {
	GameName string
	JoinedDate string
	Status string
}

type WebHandler struct {
	conf *Conf
	elo *playerElo.PlayerRanking
	oauthConf *oauth2.Config
	members []Member
	templates map[string]*template.Template
}

type OauthCreds struct {
	ClientSecret, ClientId string
}

func getLoginUrl(conf *oauth2.Config, state string) string {
	return conf.AuthCodeURL(state)
}

func fetchSpreadsheet(gid string) (Spreadsheet, error) {
	toret := Spreadsheet{}
	url := fmt.Sprintf(adSpreadsheetFmtStr, gid)
	res, err := http.Get(url)
	if err != nil {
		return toret, err
	}
	defer res.Body.Close()
	if err != nil {
		return toret, err
	}
	records := csv.NewReader(res.Body)
	toret.Headers, err = records.Read()
	if err != nil {
		return toret, err
	}
	toret.Records, err = records.ReadAll()
	return toret, err
}

func fetchCurrentMembers() ([]Member, error) {
	sheet, err := fetchSpreadsheet("0")
	if err != nil {
		return nil, err
	}
	members := []Member{}
	for _, record := range sheet.Records {
		if len(record) >= len(sheet.Headers) {
			members = append(members, Member{
				GameName : record[0],
				JoinedDate : record[1],
				Status : record[2],
			})
		}
	}
	return members, nil
}

func fetchCurrentPetitioners() (Spreadsheet, error) {
	return fetchSpreadsheet("1410959769")
}

func fetchPastMembers() (Spreadsheet, error) {
	return fetchSpreadsheet("1387129207")
}

func fetchHonoraries() (Spreadsheet, error) {
	return fetchSpreadsheet("2082464970")
}

func NewWebHandler(conf *Conf) (*WebHandler, error) {
	dbstr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", conf.Username, conf.Password, conf.Host, conf.Database)
	elo, err := playerElo.NewPlayerRankings(dbstr, 1200, 24, 10)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	templateDir := conf.TemplateDirectory
	if templateDir == "" {
		templateDir = "templates/"
	}
	templates, err := amber.CompileDir(templateDir, amber.DefaultDirOptions, amber.DefaultOptions)
	if err != nil {
		return nil, err
	}
	members, err := fetchCurrentMembers()
	if err != nil {
		return nil, err
	}
	creds := OauthCreds{}
	credFile, err := ioutil.ReadFile("./creds.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(credFile, &creds)
	if err != nil {
		return nil, err
	}
	oauthconf := &oauth2.Config {
		ClientID : creds.ClientId,
		ClientSecret : creds.ClientSecret,
		RedirectURL : "http://localhost/oauth2callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
	toret := &WebHandler{
		elo : elo,
		templates : templates,
		conf : conf,
		members : members,
		oauthConf : oauthconf,
	}
	return toret, nil
}

type Info struct {
	BasePath string
}

type Results struct {
	Info
	KitName string
	Players []playerElo.Player
}

func (wh *WebHandler) AddMatch(w http.ResponseWriter, r *http.Request) {
	kit_id := mux.Vars(r)["kit_id"]
	results, err := wh.elo.FetchByKit(kit_id)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}
	data := Results{ Info : Info{BasePath : wh.conf.EloBasePath}, Players : results, KitName : kit_id }
	wh.templates["newmatch"].Execute(w, &data)
}

func (wh *WebHandler) PostMatch(w http.ResponseWriter, r *http.Request) {
	kit_id := mux.Vars(r)["kit_id"]
	r.ParseForm()
	winner_val := r.Form.Get("winner")
	winner, err := uuid.FromString(winner_val)
	if err != nil {
		fmt.Fprintln(w, "Winner does not have a valid UUID.")
		return
	}
	loser_val := r.Form.Get("loser")
	loser, err := uuid.FromString(loser_val)
	if err != nil {
		fmt.Fprintln(w, "Loser does not have a valid UUID.")
		return
	}
	draw_val := r.Form.Get("draw")
	draw := draw_val == "draw"
	if winner != loser {
		err := wh.elo.RecordMatch(winner, loser, draw)
		if err != nil {
			fmt.Fprintln(w, err)
		}
	} else {
		fmt.Fprintln(w, "Same player entered in both fields")
		return
	}
	http.Redirect(w, r, "/elo/viewkit/" + kit_id, http.StatusSeeOther)
}

func (wh *WebHandler) AddPlayer(w http.ResponseWriter, r *http.Request) {
	data := Info{ BasePath : wh.conf.EloBasePath }
	wh.templates["newplayer"].Execute(w, &data)
}

func (wh *WebHandler) PostPlayer(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	game_name := r.Form.Get("game_name")
	realm_name := r.Form.Get("realm_name")
	kit_name := r.Form.Get("kit_name")
	gnCookie := &http.Cookie{
		Name : "game_name",
		Value : game_name,
		Path : "/",
	}
	http.SetCookie(w, gnCookie)
	rnCookie := &http.Cookie{
		Name : "realm_name",
		Value : realm_name,
		Path : "/",
	}
	http.SetCookie(w, rnCookie)
	knCookie := &http.Cookie{
		Name : "kit_name",
		Value : kit_name,
		Path : "/",
	}
	http.SetCookie(w, knCookie)
	login := getLoginUrl(wh.oauthConf, "12345678901234567890123456789012")
	http.Redirect(w, r, login, http.StatusSeeOther)
}

func (wh *WebHandler) ViewPlayer(w http.ResponseWriter, r *http.Request) {
	player_id := mux.Vars(r)["player_id"]
	fmt.Fprintln(w, "VIEWING PLAYER:", player_id)
}

func (wh *WebHandler) ViewKit(w http.ResponseWriter, r *http.Request) {
	kit_id := mux.Vars(r)["kit_id"]
	results, err := wh.elo.FetchByKit(kit_id)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	data := Results{ Info : Info{BasePath : wh.conf.EloBasePath}, Players : results, KitName : kit_id }
	wh.templates["viewkit"].Execute(w, &data)
}

func (wh *WebHandler) EloIndex(w http.ResponseWriter, r *http.Request) {
	data := Info{ BasePath : wh.conf.EloBasePath }
	wh.templates["elo_index"].Execute(w, &data)
}

type CurrentMembers struct {
	Info
	Members []Member
}

func (wh *WebHandler) CurrentMembers(w http.ResponseWriter, r *http.Request) {
	data := &CurrentMembers{
		Info : Info{ BasePath : wh.conf.EloBasePath },
		Members : wh.members,
	}
	wh.templates["currentmembers"].Execute(w, &data)
}

func (wh *WebHandler) Index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/currentmembers", http.StatusSeeOther)
}

func (wh *WebHandler) OauthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query()["code"][0]
	tok, err := wh.oauthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		panic(err)
	}
	client := wh.oauthConf.Client(oauth2.NoContext, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	email := struct{Email string `json:"email"`}{}
	json.Unmarshal(data, &email)
	game_name_cookie, _ := r.Cookie("game_name")
	realm_name_cookie, _ := r.Cookie("realm_name")
	kit_name_cookie, _ := r.Cookie("kit_name")
	if (game_name_cookie.Value != "" && realm_name_cookie.Value != "" && kit_name_cookie.Value != "") {
		if email.Email == "nolat301@gmail.com" {
			game_name := game_name_cookie.Value
			realm_name := realm_name_cookie.Value
			kit_name := kit_name_cookie.Value
			err := wh.elo.NewPlayer(game_name + ", " + realm_name, kit_name)
			if err != nil {
				fmt.Fprintln(w, err)
			}
		}
	}
	gnCookie := &http.Cookie{
		Name : "game_name",
		Value : "",
		Path : "/",
	}
	http.SetCookie(w, gnCookie)
	rnCookie := &http.Cookie{
		Name : "realm_name",
		Value : "",
		Path : "/",
	}
	http.SetCookie(w, rnCookie)
	knCookie := &http.Cookie{
		Name : "kit_name",
		Value : "",
		Path : "/",
	}
	http.SetCookie(w, knCookie)
	http.Redirect(w, r, wh.conf.EloBasePath + "/viewkit/" + kit_name_cookie.Value, http.StatusSeeOther)
}

func (wh *WebHandler) Serve() {
	router := mux.NewRouter()
	router.HandleFunc("/", wh.Index).Methods("GET")
	router.HandleFunc("/oauth2callback", wh.OauthCallback).Methods("GET")
	router.HandleFunc("/currentmembers", wh.CurrentMembers).Methods("GET")
	eloRouter := router.PathPrefix(wh.conf.EloBasePath).Subrouter()
	eloRouter.HandleFunc("/", wh.EloIndex).Methods("GET")
	eloRouter.HandleFunc("/addplayer", wh.AddPlayer).Methods("GET")
	eloRouter.HandleFunc("/addplayer", wh.PostPlayer).Methods("POST")
	eloRouter.HandleFunc("/addmatch/{kit_id}", wh.AddMatch).Methods("GET")
	eloRouter.HandleFunc("/addmatch/{kit_id}", wh.PostMatch).Methods("POST")
	eloRouter.HandleFunc("/viewplayer/{player_id}", wh.ViewPlayer).Methods("GET")
	eloRouter.HandleFunc("/viewkit/{kit_id}", wh.ViewKit).Methods("GET")
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
