# let-rds-sleep

Keep sleeping AWS RDS/Aurora Cluster.

Based on AWS Official document: https://aws.amazon.com/premiumsupport/knowledge-center/rds-stop-seven-days/?nc1=h_ls

## SYNOPSIS

```console
$ let-rds-sleep -mode STOP -target Stop=true
```

```console
$ let-rds-sleep -mode START -target Stop=true
```

## INSTALLATION

Please download binary in tarball from [releases](https://github.com/handlename/let-rds-sleep/releases).

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

Highly recommended to confirm the target resources with `-dryrun` option before execute.

```console
$ ./let-rds-sleep -mode STOP -target Sleep=true -dryrun
started as oneshot app
2023/08/18 14:56:57 [INFO] running as STOP mode
2023/08/18 14:56:58 [INFO] processing cluster/main
2023/08/18 14:56:58 [INFO] process for cluster/main is not completed [dryrun]
2023/08/18 14:56:58 [INFO] processing cluster/loadtest
2023/08/18 14:56:58 [INFO] process for cluster/loadtest is not completed [dryrun]
2023/08/18 14:56:58 [INFO] processing instance/sandbox
2023/08/18 14:56:58 [INFO] process for instance/sandbox is not completed [dryrun]
bye
```

## SETUP

This tool is supposed to run periodically as a Lambda function.
Create a function to stop/start RDS/Aurora and invoke them with EventBridge Event Rule.
Please refer to the definition examples of each AWS resource in the [terraform/example](https://github.com/handlename/let-rds-sleep/tree/main/terraform/example) directory.

## LISENCE

MIT

## AUTHOR

@handlename
