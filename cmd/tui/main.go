package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	tea "charm.land/bubbletea/v2"
	"git.rileymathews.com/riley/pr-tracker/internal/db/gen"
	"git.rileymathews.com/riley/pr-tracker/internal/db/repository"
	"git.rileymathews.com/riley/pr-tracker/internal/models"
	_ "modernc.org/sqlite"
)

type model struct {
	prs []*models.PullRequest
	cursor int
	selected map[int]struct{}
}

func initialModel(prs []*models.PullRequest) model {
	
	return model{
		prs: prs,
		cursor: 0,
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
				case "ctrl+c", "q":
					return m, tea.Quit
				case "up", "k":
					if m.cursor > 0 {
						m.cursor--
					}

				case "down", "j":
					if m.cursor < len(m.prs)-1 {
						m.cursor++
					}

				case "enter", "space":
					_, ok := m.selected[m.cursor]
					if ok {
						delete(m.selected, m.cursor)
					} else {
						m.selected[m.cursor] = struct{}{}
					}
			}
	}

	return m, nil
}

func (m model) View() tea.View {
	s := "What should we buy at the market?\n\n"

	for i, choice := range m.prs {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += "\n Press q to quit.\n"

	return tea.NewView(s)
}

func main() {
	dbConn, err := sql.Open("sqlite", "./db.sqlite3")
	if err != nil {
		log.Fatalf("open sqlite db failed: %v", err)
	}
	defer func() {
		if closeErr := dbConn.Close(); closeErr != nil {
			log.Printf("close sqlite db failed: %v", closeErr)
		}
	}()

	ctx := context.Background()
	if err := repository.ApplyMigrations(ctx, dbConn, "internal/db/migrations"); err != nil {
		log.Fatalf("apply sqlite migrations failed: %v", err)
	}

	queries := gen.New(dbConn)
	repo := repository.New(queries, ctx)

	prs, err := repo.GetAllPrs()
	if err != nil {
		log.Fatalf("could not fetch PRs %v", err)
	}

	p := tea.NewProgram(initialModel(prs))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas there's been an error: %v", err)
		os.Exit(1)
	}
}
