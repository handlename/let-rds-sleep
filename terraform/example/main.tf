terraform {
  required_version = "1.7.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5"
    }
  }

  backend "local" {}
}

provider "aws" {}

resource "aws_iam_role" "scheduler" {
  name = "let-rds-sleep-scheduler"
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Action = "sts:AssumeRole",
        Effect = "Allow",
        Principal = {
          Service = "scheduler.amazonaws.com",
        },
      },
    ],
  })
}

resource "aws_iam_role_policy" "scheduler" {
  name   = "let-rds-sleep-scheduler"
  role   = aws_iam_role.scheduler.id
  policy = data.aws_iam_policy_document.scheduler.json
}

data "aws_iam_policy_document" "scheduler" {
  version = "2012-10-17"

  statement {
    effect = "Allow"

    actions = [
      "lambda:InvokeFunction",
    ]

    resources = [
      aws_lambda_function.let-rds-sleep.arn,
    ]
  }
}

resource "aws_scheduler_schedule" "schedule" {
  for_each = {
    start = "cron(7 6 ? * MON *)", # before maintenance window
    stop  = "cron(7 7 ? * MON *)"  # after maintenance window
  }

  name       = "let-rds-sleep-${each.key}"
  group_name = "default"

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression = each.value

  target {
    arn      = aws_lambda_function.let-rds-sleep.arn
    role_arn = aws_iam_role.scheduler.arn

    input = jsonencode({
      mode : each.key,
    })
  }
}

resource "aws_iam_role" "lambda-let-rds-sleep" {
  name               = "let-rds-sleep"
  assume_role_policy = data.aws_iam_policy_document.assume-role-policy.json
}

resource "aws_iam_role_policy" "lambda-let-rds-sleep" {
  name   = "let-rds-sleep"
  role   = aws_iam_role.lambda-let-rds-sleep.id
  policy = data.aws_iam_policy_document.lambda-let-rds-sleep.json
}

data "aws_iam_policy_document" "assume-role-policy" {
  version = "2012-10-17"

  statement {
    actions = [
      "sts:AssumeRole",
    ]

    effect = "Allow"

    principals {
      type = "Service"

      identifiers = [
        "lambda.amazonaws.com",
      ]
    }
  }
}

data "aws_iam_policy_document" "lambda-let-rds-sleep" {
  version = "2012-10-17"

  statement {
    effect = "Allow"

    actions = [
      "rds:Describe*",
      "rds:StartDB*",
      "rds:StopDB*",
    ]

    resources = [
      "*",
    ]
  }
}

resource "aws_iam_role_policy_attachment" "lambda-let-rds-sleep-lambda-basic-execution-role" {
  role       = aws_iam_role.lambda-let-rds-sleep.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

data "archive_file" "let-rds-sleep-src" {
  type        = "zip"
  source_file = "bootstrap" # renamed let-rds-sleep binary
  output_path = "lambda_function_payload.zip"
}

resource "aws_cloudwatch_log_group" "lambda" {
  name = "/aws/lambda/let-rds-sleep-start"
}

resource "aws_lambda_function" "let-rds-sleep" {
  filename      = "lambda_function_payload.zip"
  function_name = "let-rds-sleep"
  role          = aws_iam_role.lambda-let-rds-sleep.arn
  handler       = "bootstrap"

  source_code_hash = data.archive_file.let-rds-sleep-src.output_base64sha256

  runtime = "provided.al2"

  environment {
    variables = {
      LET_RDS_SLEEP_LOG_LEVEL = "INFO"
      LET_RDS_SLEEP_TARGET    = "Sleep=true"
      LET_RDS_SLEEP_DRYRUN    = "1"
      LOG_LEVEL               = "DEBUG"
    }
  }
}
