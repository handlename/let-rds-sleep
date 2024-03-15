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

resource "aws_cloudwatch_event_rule" "start" {
  name                = "let-rds-sleep-start"
  schedule_expression = "cron(7 6 ? * MON *)" # before maintenance window
}

resource "aws_cloudwatch_event_rule" "stop" {
  name                = "let-rds-sleep-stop"
  schedule_expression = "cron(7 7 ? * MON *)" # after maintenance window
}

resource "aws_cloudwatch_event_target" "start" {
  arn  = aws_lambda_function.let-rds-sleep.arn
  rule = aws_cloudwatch_event_rule.start.name

  input = jsonencode({
    mode : "start",
  })
}

resource "aws_cloudwatch_event_target" "stop" {
  arn  = aws_lambda_function.let-rds-sleep.arn
  rule = aws_cloudwatch_event_rule.stop.name

  input = jsonencode({
    mode : "stop",
  })
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
  name              = "/aws/lambda/let-rds-sleep-start"
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
    }
  }
}

resource "aws_lambda_permission" "permission-start" {
  source_arn    = aws_cloudwatch_event_rule.start.arn
  function_name = aws_lambda_function.let-rds-sleep.function_name
  action        = "lambda:InvokeFunction"
  principal     = "events.amazonaws.com"
}

resource "aws_lambda_permission" "permission-stop" {
  source_arn    = aws_cloudwatch_event_rule.stop.arn
  function_name = aws_lambda_function.let-rds-sleep.function_name
  action        = "lambda:InvokeFunction"
  principal     = "events.amazonaws.com"
}
