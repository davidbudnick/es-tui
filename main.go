package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidbudnick/es-tui/internal/cmd"
	"github.com/davidbudnick/es-tui/internal/db"
	"github.com/davidbudnick/es-tui/internal/es"
	"github.com/davidbudnick/es-tui/internal/service"
	"github.com/davidbudnick/es-tui/internal/types"
	"github.com/davidbudnick/es-tui/internal/ui"

	tea "charm.land/bubbletea/v2"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Overridable in tests.
var (
	osExit   = os.Exit
	logFatal = prodLogFatal
	runApp   = prodRunApp
)

// Overridable for tests (log.Fatal never returns).
var logFatalf = func(v ...any) { log.Fatal(v...) }

func prodLogFatal(v ...any) { logFatalf(v...) }

// newProgram is overridable so prodRunApp can be covered without a TTY.
var newProgram = func(m ui.Model) teaProgram {
	p := tea.NewProgram(m)
	return p
}

type teaProgram interface {
	Send(msg tea.Msg)
	Run() (tea.Model, error)
}

func prodRunApp(m ui.Model) error {
	p := newProgram(m)
	if m.SendFunc != nil {
		*m.SendFunc = p.Send
	}
	_, err := p.Run()
	return err
}

func main() {
	m, err := setup()
	if err != nil {
		logFatal(err)
		return
	}
	if err := runApp(m); err != nil {
		logFatal(err)
	}
}

func setup() (ui.Model, error) {
	opts := parseCLIFlags()

	logWriter := types.NewLogWriter()

	m := ui.NewModel()
	m.Logs = logWriter
	m.Version = version

	if opts != nil {
		m.CLIConnection = opts
	}

	sendFunc := func(msg tea.Msg) {}
	m.SendFunc = &sendFunc

	handler := slog.NewJSONHandler(logWriter, nil)
	slog.SetDefault(slog.New(handler))

	config, err := initConfig()
	if err != nil {
		return m, fmt.Errorf("failed to initialize config: %w", err)
	}

	esClient := es.NewClient()
	container := &service.Container{Config: config, ES: esClient}
	m.Cmds = cmd.NewCommandsFromContainer(container)

	return m, nil
}

func parseCLIFlags() *types.Connection {
	conn, showVersion, doUpdate, err := parseFlags(os.Args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			osExit(0)
			return nil
		}
		osExit(2)
		return nil
	}
	if showVersion {
		fmt.Printf("es-tui %s (commit: %s, built: %s)\n", version, commit, date)
		osExit(0)
		return nil
	}
	if doUpdate {
		if err := runUpdate(version); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			osExit(1)
			return nil
		}
		osExit(0)
		return nil
	}
	return conn
}

func parseFlags(args []string) (conn *types.Connection, showVersion bool, doUpdate bool, err error) {
	fs := flag.NewFlagSet("es-tui", flag.ContinueOnError)

	host := fs.String("host", "", "Elasticsearch/OpenSearch hostname (required for quick-connect)")
	port := fs.Int("port", 9200, "Server port")
	username := fs.String("user", "", "Username for basic auth")
	password := fs.String("password", "", "Password for basic auth")
	apiKey := fs.String("api-key", "", "API key authentication")
	name := fs.String("name", "", "Connection display name")
	flavor := fs.String("flavor", "auto", "Engine flavor: auto, elasticsearch, opensearch")
	tls := fs.Bool("tls", false, "Enable TLS/SSL")
	tlsCert := fs.String("tls-cert", "", "TLS client certificate file")
	tlsKey := fs.String("tls-key", "", "TLS client private key file")
	tlsCA := fs.String("tls-ca", "", "TLS CA certificate file")
	tlsSkipVerify := fs.Bool("tls-skip-verify", false, "Skip TLS certificate verification")
	versionFlag := fs.Bool("version", false, "Print version and exit")
	update := fs.Bool("update", false, "Update to the latest version")

	fs.StringVar(host, "h", "", "Hostname (shorthand)")
	fs.IntVar(port, "p", 9200, "Port (shorthand)")
	fs.StringVar(password, "a", "", "Password (shorthand)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: es-tui [flags]\n\n")
		fmt.Fprintf(os.Stderr, "A terminal UI for Elasticsearch and OpenSearch.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fmt.Fprintf(os.Stderr, "  -h, --host string       Server hostname (required for quick-connect)\n")
		fmt.Fprintf(os.Stderr, "  -p, --port int          Server port (default 9200)\n")
		fmt.Fprintf(os.Stderr, "  -a, --password string   Password for basic auth\n")
		fmt.Fprintf(os.Stderr, "      --user string       Username for basic auth\n")
		fmt.Fprintf(os.Stderr, "      --api-key string    API key authentication\n")
		fmt.Fprintf(os.Stderr, "      --name string       Connection display name\n")
		fmt.Fprintf(os.Stderr, "      --flavor string     auto|elasticsearch|opensearch (default auto)\n")
		fmt.Fprintf(os.Stderr, "      --tls               Enable TLS/SSL\n")
		fmt.Fprintf(os.Stderr, "      --tls-cert string   TLS client certificate file\n")
		fmt.Fprintf(os.Stderr, "      --tls-key string    TLS client private key file\n")
		fmt.Fprintf(os.Stderr, "      --tls-ca string     TLS CA certificate file\n")
		fmt.Fprintf(os.Stderr, "      --tls-skip-verify   Skip TLS certificate verification\n")
		fmt.Fprintf(os.Stderr, "      --version           Print version and exit\n")
		fmt.Fprintf(os.Stderr, "      --update            Update to the latest version\n")
	}

	if err := fs.Parse(args); err != nil {
		return nil, false, false, err
	}

	if *versionFlag {
		return nil, true, false, nil
	}
	if *update {
		return nil, false, true, nil
	}
	if *host == "" {
		return nil, false, false, nil
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "password" || f.Name == "a" || f.Name == "api-key" {
			fmt.Fprintln(os.Stderr, "Warning: Passing secrets on the command line exposes them in the process list. Prefer the interactive connection form.")
		}
	})

	fl := types.Flavor(strings.ToLower(*flavor))
	switch fl {
	case types.FlavorElasticsearch, types.FlavorOpenSearch, types.FlavorAuto:
	default:
		fl = types.FlavorAuto
	}

	conn = &types.Connection{
		Host:     *host,
		Port:     *port,
		Username: *username,
		Password: *password,
		APIKey:   *apiKey,
		Flavor:   fl,
		UseTLS:   *tls,
	}
	if *name != "" {
		conn.Name = *name
	} else {
		conn.Name = fmt.Sprintf("%s:%d", *host, *port)
	}
	if *tls {
		conn.TLSConfig = &types.TLSConfig{
			CertFile:           *tlsCert,
			KeyFile:            *tlsKey,
			CAFile:             *tlsCA,
			InsecureSkipVerify: *tlsSkipVerify,
		}
	}
	return conn, false, false, nil
}

// Overridable in tests.
var userHomeDir = os.UserHomeDir

func initConfig() (*db.Config, error) {
	homeDir, err := userHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}

	configDir := filepath.Join(homeDir, ".config", "es-tui")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")
	return db.NewConfig(configPath)
}
