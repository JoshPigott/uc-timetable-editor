[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$InputPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$branchTypes = @(
    'feat'
    'fix'
    'test'
    'ci'
    'docs'
    'refactor'
    'perf'
    'chore'
    'build'
)

$commitPattern = '^(?<type>[a-z]+)(?:\((?<scope>[^()]+)\))?(?<breaking>!)?: (?<subject>.+)$'

function Test-ConventionalCommit {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message,

        [Parameter(Mandatory = $true)]
        [string[]]$AllowedTypes
    )

    if ($Message -cnotmatch $commitPattern) {
        return "must follow '<type>: <subject>' (for example 'feat: add favicon')"
    }

    $type = $Matches['type']
    if ($type -notin $AllowedTypes) {
        return "uses type '$type', which is not one of: $($AllowedTypes -join ', ')"
    }

    if ($Matches['subject'] -cnotmatch '^\S') {
        return 'has an empty subject'
    }

    return $null
}

if (-not (Test-Path -LiteralPath $InputPath)) {
    throw "Pull request metadata file not found: $InputPath"
}

$payload = Get-Content -LiteralPath $InputPath -Raw | ConvertFrom-Json

$failures = [System.Collections.Generic.List[string]]::new()

$branchMatch = [regex]::Match($payload.branch, '^(?<type>[a-z]+)/.+')
if (-not $branchMatch.Success) {
    $failures.Add("branch '$($payload.branch)' must start with a type prefix followed by '/', for example 'feat/short-description'")
} elseif ($branchMatch.Groups['type'].Value -notin $branchTypes) {
    $failures.Add("branch '$($payload.branch)' uses type '$($branchMatch.Groups['type'].Value)', which is not one of: $($branchTypes -join ', ')")
}

$titleFailure = Test-ConventionalCommit -Message $payload.title -AllowedTypes $branchTypes
if ($titleFailure) {
    $failures.Add("pull request title '$($payload.title)' $titleFailure")
}

$commits = @($payload.commits)
if ($commits.Count -eq 0) {
    $failures.Add('pull request contains no commits to validate')
}

foreach ($commit in $commits) {
    $commitFailure = Test-ConventionalCommit -Message $commit -AllowedTypes $branchTypes
    if ($commitFailure) {
        $failures.Add("commit '$commit' $commitFailure")
    }
}

if ($failures.Count -gt 0) {
    Write-Host 'Pull request conventions failed:' -ForegroundColor Red
    foreach ($failure in $failures) {
        Write-Host "  - $failure" -ForegroundColor Red
    }
    Write-Host ''
    Write-Host "Allowed branch prefixes: $($branchTypes -join '/')/"
    Write-Host "Commit and title format: <type>: <subject>, for example 'fix: exclude past events from the feed'"
    exit 1
}

Write-Host 'Pull request conventions passed.' -ForegroundColor Green
exit 0