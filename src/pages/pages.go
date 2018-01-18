package pages

import (
	"playerElo"
	"html/template"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/eknkc/amber"
	"github.com/satori/go.uuid"
	"fmt"
)

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

type WebHandler struct {
	conf *Conf
	elo *playerElo.PlayerRanking
	templates map[string]*template.Template
}

func NewWebHandler(conf *Conf) (*WebHandler, error) {
	dbstr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", conf.Username, conf.Password, conf.Host, conf.Database)
	elo, err := playerElo.NewPlayerRankings(dbstr, 1200, 24, 10)
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
	toret := &WebHandler{
		elo : elo,
		templates : templates,
		conf : conf,
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

func (wh *WebHandler) Serve() {
	fmt.Println(wh.conf)
	router := mux.NewRouter().PathPrefix(wh.conf.EloBasePath).Subrouter()
	router.HandleFunc("/", wh.EloIndex).Methods("GET")
	router.HandleFunc("/addplayer", wh.AddPlayer).Methods("GET")
	router.HandleFunc("/addmatch/{kit_id}", wh.AddMatch).Methods("GET")
	router.HandleFunc("/viewplayer/{player_id}", wh.ViewPlayer).Methods("GET")
	router.HandleFunc("/viewkit/{kit_id}", wh.ViewKit).Methods("GET")
	http.Handle("/", router)
	http.ListenAndServe(":8181", nil)
}
