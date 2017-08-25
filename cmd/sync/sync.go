package main

import (
	"fmt"
	lib "k8s.io/test-infra/gha2db"
	"os"
	"strconv"
	"strings"
	"time"
)

func sync(args []string) {
	// Environment context parse
	var ctx lib.Ctx
	ctx.Init()

	// Strip function to be used by MapString
	stripFunc := func(x string) string { return strings.TrimSpace(x) }

	// Orgs & Repos
	sOrg := ""
	if len(args) > 0 {
		sOrg = args[0]
	}
	sRepo := ""
	if len(args) > 1 {
		sRepo = args[1]
	}
	org := lib.StringsMapToArray(stripFunc, strings.Split(sOrg, ","))
	repo := lib.StringsMapToArray(stripFunc, strings.Split(sRepo, ","))
	fmt.Printf("Running on: %s/%s\n", strings.Join(org, "+"), strings.Join(repo, "+"))

	// Connect to Postgres DB
	con := lib.PgConn(&ctx)
	defer con.Close()

	// Connect to InfluxDB
	ic, _ := lib.IDBConn(&ctx)
	defer ic.Close()

	// Get max event date from Postgres database
	var maxDtPtr *time.Time
	maxDtPg := ctx.DefaultStartDate
	lib.FatalOnError(lib.QueryRowSQL(con, &ctx, "select max(created_at) from gha_events").Scan(&maxDtPtr))
	if maxDtPtr != nil {
		maxDtPg = *maxDtPtr
	}

	// Get max series date from Influx database
	maxDtIDB := ctx.DefaultStartDate
	res := lib.QueryIDB(ic, &ctx, "select last(value) from "+ctx.LastSeries)
	series := res[0].Series
	if len(series) > 0 {
		maxDtIDB = lib.TimeParseIDB(series[0].Values[0][0].(string))
	}
	fmt.Printf("maxDtPg = %v, maxDtIDB = %v\n", maxDtPg, maxDtIDB)

	// Create date range
	// Just to get into next GHA hour
	from := maxDtPg.Add(5 * time.Minute)
	to := time.Now()
	fromDate := lib.ToYMDDate(from)
	fromHour := strconv.Itoa(from.Hour())
	toDate := lib.ToYMDDate(to)
	toHour := strconv.Itoa(to.Hour())

	// Get new GHAs
	fmt.Printf("Range: %s %s - %s %s\n", fromDate, fromHour, toDate, toHour)
	lib.ExecCommand(
		&ctx,
		[]string{
			"./gha2db",
			fromDate,
			fromHour,
			toDate,
			toHour,
			strings.Join(org, ","),
			strings.Join(repo, ","),
		},
		nil,
	)
	fmt.Printf("we're fine.\n")
	/*
	  res = system cmd
	  unless res
	    puts "Command failed: '#{cmd}'"
	    exit 1
	  end

	  # Recompute views and DB summaries
	  cmd = './structure.rb'
	  puts cmd
	  res = system(
	    { 'GHA2DB_SKIPTABLE' => '1', 'GHA2DB_MGETC' => 'y' },
	    cmd
	  )
	  unless res
	    puts "Command failed: '#{cmd}'"
	    exit 1
	  end

	  # DB2Influx
	  unless ENV['GHA2DB_SKIPIDB']
	    metrics_dir = $pg ? 'psql_metrics' : 'mysql_metrics'
	    from = Time.parse('2015-08-01').utc if ENV['GHA2DB_RESETIDB']

	    # Reviewers daily, weekly, monthly, quarterly, yearly
	    %w[d w m q y].each do |period|
	      cmd = "./db2influx.rb reviewers_#{period} #{metrics_dir}/reviewers.sql "\
	            "'#{to_ymd(from)}' '#{to_ymd(to)}' #{period}"
	      puts cmd
	      res = system cmd
	      unless res
	        puts "Command failed: '#{cmd}'"
	        exit 1
	      end
	    end

	    # SIG mentions daily, weekly, monthly, quarterly, yearly
	    %w[d w m q y].each do |period|
	      cmd = "./db2influx.rb sig_mentions_data #{metrics_dir}/sig_mentions.sql "\
	            "'#{to_ymd(from)}' '#{to_ymd(to)}' #{period}"
	      puts cmd
	      res = system cmd
	      unless res
	        puts "Command failed: '#{cmd}'"
	        exit 1
	      end
	    end

	    # PRs merged per repo daily, weekly, monthly, quarterly, yearly
	    %w[d w m q y].each do |period|
	      cmd = "./db2influx.rb prs_merged_data #{metrics_dir}/prs_merged.sql "\
	            "'#{to_ymd(from)}' '#{to_ymd(to)}' #{period}"
	      puts cmd
	      res = system cmd
	      unless res
	        puts "Command failed: '#{cmd}'"
	        exit 1
	      end
	    end

	    # All PRs merged hourly, daily, weekly, monthly, quarterly, yearly
	    %w[h d w m q y].each do |period|
	      cmd = "./db2influx.rb all_prs_merged_#{period} #{metrics_dir}/all_prs_merged.sql "\
	            "'#{to_ymdhms(from)}' '#{to_ymdhms(to)}' #{period}"
	      puts cmd
	      res = system cmd
	      unless res
	        puts "Command failed: '#{cmd}'"
	        exit 1
	      end
	    end

	    # Time opened to merged (number of hours) daily, weekly, monthly, quarterly, yearly
	    %w[d w m q y].each do |period|
	      cmd = "./db2influx.rb hours_pr_open_to_merge_#{period} #{metrics_dir}/opened_to_merged.sql "\
	            "'#{to_ymd(from)}' '#{to_ymd(to)}' #{period}"
	      puts cmd
	      res = system cmd
	      unless res
	        puts "Command failed: '#{cmd}'"
	        exit 1
	      end
	    end
	  end

	  puts 'Sync success'
	*/
}

func main() {
	sync(os.Args[1:])
}