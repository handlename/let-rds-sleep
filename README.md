# let-rds-sleep

Keep sleeping AWS RDS/Aurora Cluster.

Based on AWS Official document: https://aws.amazon.com/premiumsupport/knowledge-center/rds-stop-seven-days/?nc1=h_ls

## SYNOPSIS

```console
$ let-rds-sleep -mode STOP -target Stop=true
```

## USAGE

```console
$ let-rds-sleep -help
Usage of ./let-rds-sleep:
  -dryrun
    	show process target only
  -exclude string
    	TagName=Value,... If Tag exists exclude the resource
  -mode string
    	STOP or START
  -target string
    	TagName=Value,... If no tags given, treat all of resources as target
  -version
    	display version
```

## LISENCE

MIT

## AUTHOR

@handlename
