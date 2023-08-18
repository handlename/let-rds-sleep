terraform {
  required_version = "1.5.2"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.1.0"
    }
  }

  backend "local" {}
}

provider "aws" {}

resource "aws_cloudwatch_event_rule" "start" {
  name                = "let-rds-sleep-start"
  schedule_expression = "cron(7 6 ? * MON *)" # before maintenance window
}

resource "aws_cloudwatch_event_rule" "stop" {
  name                = "let-rds-sleep-stop"
  schedule_expression = "cron(7 7 ? * MON *)" # after maintenance window
}

resource "aws_cloudwatch_event_target" "start" {
  arn  = aws_lambda_function.let-rds-sleep-start.arn
  rule = aws_cloudwatch_event_rule.start.name
}

resource "aws_cloudwatch_event_target" "stop" {
  arn  = aws_lambda_function.let-rds-sleep-stop.arn
  rule = aws_cloudwatch_event_rule.stop.name
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

resource "aws_lambda_function" "let-rds-sleep-start" {
  filename      = "lambda_function_payload.zip"
  function_name = "let-rds-sleep-start"
  role          = aws_iam_role.lambda-let-rds-sleep.arn
  handler       = "bootstrap"

  source_code_hash = data.archive_file.let-rds-sleep-src.output_base64sha256

  runtime = "provided.al2"

  environment {
    variables = {
      LET_RDS_SLEEP_LOG_LEVEL = "INFO"
      LET_RDS_SLEEP_MODE      = "START",
      LET_RDS_SLEEP_TARGET    = "Sleep=true"
    }
  }
}

resource "aws_lambda_function" "let-rds-sleep-stop" {
  filename      = "lambda_function_payload.zip"
  function_name = "let-rds-sleep-stop"
  role          = aws_iam_role.lambda-let-rds-sleep.arn
  handler       = "bootstrap"

  source_code_hash = data.archive_file.let-rds-sleep-src.output_base64sha256

  runtime = "provided.al2"

  environment {
    variables = {
      LET_RDS_SLEEP_LOG_LEVEL = "INFO"
      LET_RDS_SLEEP_MODE      = "STOP",
      LET_RDS_SLEEP_TARGET    = "Sleep=true"
    }
  }
}
