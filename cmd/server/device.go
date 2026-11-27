package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/auth"
	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	authstore "github.com/tiagorocha94/household-webapp/internal/auth/store"
	"github.com/tiagorocha94/household-webapp/internal/config"
	"github.com/tiagorocha94/household-webapp/internal/database"
)

const (
	// deviceCommands lists the device commands, for the usage of both the
	// server's own help and the device command.
	deviceCommands = `  household-app device add [-config file] <user> <name>   make an enrolment code for a phone
  household-app device list [-config file]                 list devices
  household-app device revoke [-config file] <name>        stop a device from working
`
	deviceUsage = "usage:\n" + deviceCommands
)

// runDevice runs the "device" command: args are what follows the word
// "device". It manages the phones that write data, working on the database
// directly, so it needs no running server (the server has to have started once
// so the database and the users exist).
func runDevice(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing device command\n%s", deviceUsage)
	}
	sub, rest := args[0], args[1:]

	fs := flag.NewFlagSet("device "+sub, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	configPath := fs.String("config", "config.yaml", "path to the configuration file")
	if err := fs.Parse(rest); err != nil {
		return fmt.Errorf("%w\n%s", err, deviceUsage)
	}
	operands := fs.Args()

	var wantOperands int
	switch sub {
	case "add":
		wantOperands = 2
	case "revoke":
		wantOperands = 1
	case "list":
		wantOperands = 0
	default:
		return fmt.Errorf("unknown device command %q\n%s", sub, deviceUsage)
	}
	if len(operands) != wantOperands {
		return fmt.Errorf("device %s takes %d argument(s), got %d\n%s", sub, wantOperands, len(operands), deviceUsage)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if _, err := os.Stat(cfg.Database.Path); err != nil {
		return fmt.Errorf("database %s not found: start the server once first: %w", cfg.Database.Path, err)
	}
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close() }()
	svc := authservice.New(authstore.New(db))

	switch sub {
	case "add":
		return deviceAdd(stdout, svc, cfg, operands[0], operands[1])
	case "revoke":
		return deviceRevoke(stdout, svc, operands[0])
	default:
		return deviceList(stdout, svc, time.Now())
	}
}

func deviceAdd(w io.Writer, svc *authservice.Service, cfg *config.Config, username, name string) error {
	code, expires, err := svc.AddDevice(username, name)
	switch {
	case errors.Is(err, auth.ErrUnknownUser):
		return fmt.Errorf("no user %q: users come from users.csv, and the server imports them when it starts", username)
	case errors.Is(err, auth.ErrUserHasDevice):
		return fmt.Errorf("%s already has a device: revoke it first with \"device revoke <name>\" (see \"device list\")", username)
	case errors.Is(err, auth.ErrDeviceNameTaken):
		return fmt.Errorf("a device named %q already exists: pick another name", name)
	case err != nil:
		return err
	}

	_, _ = fmt.Fprintf(w, "Enrolment code for %s's device %q:\n\n    %s\n\n", username, strings.TrimSpace(name), code)
	_, _ = fmt.Fprintf(w, "Type it in the app within %d minutes (until %s). It works once.\n", int(authservice.EnrolCodeTTL/time.Minute), expires.Format("15:04"))
	if cfg.Server.PublicURL != "" {
		_, _ = fmt.Fprintf(w, "Server address for the app: %s\n", cfg.Server.PublicURL)
	} else {
		_, _ = fmt.Fprintln(w, "Server address for the app: set server.public_url in the configuration to have it shown here.")
	}
	return nil
}

func deviceRevoke(w io.Writer, svc *authservice.Service, name string) error {
	if err := svc.RevokeDevice(name); err != nil {
		if errors.Is(err, auth.ErrUnknownDevice) {
			return fmt.Errorf("no active device named %q (see \"device list\")", name)
		}
		return err
	}
	_, _ = fmt.Fprintf(w, "Revoked %q. It can no longer write; its person can be given a new device.\n", name)
	return nil
}

func deviceList(w io.Writer, svc *authservice.Service, now time.Time) error {
	devices, err := svc.ListDevices()
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		_, _ = fmt.Fprintln(w, "No devices. Make one with \"device add <user> <name>\".")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "NAME\tUSER\tSTATUS\tLAST SEEN")
	for _, d := range devices {
		lastSeen := "never"
		if !d.LastSeenAt.IsZero() {
			lastSeen = d.LastSeenAt.Local().Format("2006-01-02 15:04")
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", d.Name, d.Username, d.Status(now), lastSeen)
	}
	return tw.Flush()
}
