package web

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"carsharing/internal/models"
)

type Page struct {
	Path  string
	Name  string
	Title string
	Roles []string
	Menu  bool
}

const RoleGuest = "guest"

var (
	staff = []string{models.RoleManager, models.RoleAdmin}
	all   = []string{models.RoleClient, models.RoleManager, models.RoleAdmin}
)

var Pages = []Page{
	{Path: "/", Name: "home", Title: "Главная"},
	{Path: "/login", Name: "login", Title: "Вход", Roles: []string{RoleGuest}, Menu: true},
	{Path: "/register", Name: "register", Title: "Регистрация", Roles: []string{RoleGuest}, Menu: true},
	{Path: "/catalog", Name: "catalog", Title: "Каталог и карта", Roles: all, Menu: true},
	{Path: "/rentals/my", Name: "my-rentals", Title: "Мои аренды", Roles: []string{models.RoleClient}, Menu: true},
	{Path: "/fleet", Name: "fleet", Title: "Автомобили", Roles: staff, Menu: true},
	{Path: "/tariffs", Name: "tariffs", Title: "Тарифы", Roles: staff, Menu: true},
	{Path: "/maintenance", Name: "maintenance", Title: "Обслуживание", Roles: staff, Menu: true},
	{Path: "/rentals", Name: "rentals", Title: "Все аренды", Roles: staff, Menu: true},
	{Path: "/stats", Name: "stats", Title: "Статистика", Roles: staff, Menu: true},
	{Path: "/users", Name: "users", Title: "Пользователи и роли", Roles: []string{models.RoleAdmin}, Menu: true},
	{Path: "/about", Name: "about", Title: "Об авторе", Menu: true},
	{Path: "/profile", Name: "profile", Title: "Профиль", Roles: all},
}

type pageData struct {
	Page   Page
	Roles  string
	Script string
	Menu   []menuItem
	Year   int
	Status int
}

type menuItem struct {
	Path, Title, Roles string
}

type Handler struct {
	mux   *http.ServeMux
	pages map[string]*template.Template
}

var notFoundPage = Page{Name: "not-found", Title: "Страница не найдена"}

func New(dir string) (*Handler, error) {
	tplDir := filepath.Join(dir, "templates")
	staticDir := filepath.Join(dir, "static")
	if st, err := os.Stat(staticDir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("не найден каталог клиентской части %s (задайте FRONTEND_DIR)", staticDir)
	}

	h := &Handler{mux: http.NewServeMux(), pages: map[string]*template.Template{}}
	layout := filepath.Join(tplDir, "layout.html")

	var menu []menuItem
	for _, p := range Pages {
		if p.Menu {
			menu = append(menu, menuItem{Path: p.Path, Title: p.Title, Roles: strings.Join(p.Roles, " ")})
		}
	}

	pages := append(append([]Page{}, Pages...), notFoundPage)
	for _, p := range pages {
		tpl, err := template.ParseFiles(layout, filepath.Join(tplDir, "pages", p.Name+".html"))
		if err != nil {
			return nil, fmt.Errorf("шаблон %s: %w", p.Name, err)
		}
		h.pages[p.Name] = tpl
		if p.Name == notFoundPage.Name {
			continue
		}
		data := pageData{Page: p, Roles: strings.Join(p.Roles, " "), Menu: menu, Status: http.StatusOK}
		if _, err := os.Stat(filepath.Join(staticDir, "js", "pages", p.Name+".js")); err == nil {
			data.Script = "/static/js/pages/" + p.Name + ".js"
		}
		pattern := "GET " + p.Path
		if p.Path == "/" {
			pattern = "GET /{$}"
		}
		h.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) { h.render(w, data) })
	}

	notFound := pageData{Page: notFoundPage, Menu: menu, Status: http.StatusNotFound}
	h.mux.Handle("GET /static/", http.StripPrefix("/static/", staticFiles(http.FileServer(http.Dir(staticDir)))))
	h.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			probe := r.Clone(r.Context())
			probe.Method = http.MethodGet
			if _, pattern := h.mux.Handler(probe); pattern != "/" {
				w.Header().Set("Allow", "GET, HEAD")
				http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
				return
			}
		}
		h.render(w, notFound)
	})
	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) render(w http.ResponseWriter, data pageData) {
	data.Year = time.Now().Year()
	var buf bytes.Buffer
	if err := h.pages[data.Page.Name].ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("render page %s: %v", data.Page.Name, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(data.Status)
	w.Write(buf.Bytes())
}

const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self'; " +
	"img-src 'self' data: https://tiles.stadiamaps.com https://tiles.openfreemap.org; " +
	"connect-src 'self' https://tiles.openfreemap.org; worker-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

func staticFiles(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
