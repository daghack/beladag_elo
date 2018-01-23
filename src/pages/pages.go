package pages

import (
	"encoding/csv"
	"playerElo"
	"html/template"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/eknkc/amber"
	"github.com/satori/go.uuid"
	"fmt"
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
	members []Member
	templates map[string]*template.Template
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
	toret := &WebHandler{
		elo : elo,
		templates : templates,
		conf : conf,
		members : members,
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
	http.Redirect(w, r, "/elo/rank/" + kit_id, http.StatusSeeOther)
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
	err := wh.elo.NewPlayer(game_name + ", " + realm_name, kit_name)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	http.Redirect(w, r, "/elo/viewkit/" + kit_name, http.StatusSeeOther)
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

func (wh *WebHandler) Serve() {
	router := mux.NewRouter()
	router.HandleFunc("/", wh.Index).Methods("GET")
	router.HandleFunc("/currentmembers", wh.CurrentMembers).Methods("GET")
	eloRouter := router.PathPrefix(wh.conf.EloBasePath).Subrouter()
	eloRouter.HandleFunc("/", wh.EloIndex).Methods("GET")
	eloRouter.HandleFunc("/addplayer", wh.AddPlayer).Methods("GET")
	eloRouter.HandleFunc("/addmatch/{kit_id}", wh.AddMatch).Methods("GET")
	eloRouter.HandleFunc("/viewplayer/{player_id}", wh.ViewPlayer).Methods("GET")
	eloRouter.HandleFunc("/viewkit/{kit_id}", wh.ViewKit).Methods("GET")
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
