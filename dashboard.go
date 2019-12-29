package randomize

import (
	"log"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Dashboard(w http.ResponseWriter, r *http.Request) {

	log.Printf("Dashboard")

	if r.Method != "GET" {
		log.Printf("Dashboard method != GET")
		Serve404(w)
		return
	}

	ctx := r.Context()
	useremail := userEmail(r)
	log.Printf("Dashboard email=%s", useremail)

	projlist, err := getProjects(ctx, useremail, true)
	if status.Code(err) == codes.NotFound {
		// OK
	} else if err != nil {
		msg := "A database error occured, projects cannot be retrieved."
		log.Printf("Dashboard: %v", err)
		rmsg := "Return to dashboard"
		messagePage(w, r, msg, rmsg, "/dashboard")
		return
	}

	tvals := struct {
		User     string
		LoggedIn bool
		PRN      bool
		PR       []*ProjectView
	}{
		User:     useremail,
		PR:       formatProjects(projlist),
		PRN:      len(projlist) > 0,
		LoggedIn: useremail != "",
	}

	if err := tmpl.ExecuteTemplate(w, "dashboard.html", tvals); err != nil {
		log.Printf("Dashboard failed to execute template: %v", err)
	}
}
