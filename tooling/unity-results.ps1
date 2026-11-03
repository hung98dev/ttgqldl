# Shared NUnit result evaluation for the Unity test wrappers.
#
# Dot-source this file. A run that produced zero executed tests, or a suite where
# every test was skipped, is a failure: an empty XML file must never be reported
# as a passing platform.

function Get-UnityTestSummary {
    param([Parameter(Mandatory = $true)][string]$XmlPath)

    if (-not (Test-Path $XmlPath)) {
        throw "result file not found: $XmlPath"
    }

    [xml]$document = Get-Content -Path $XmlPath
    $run = $document.'test-run'
    if ($null -eq $run) {
        throw "result file has no <test-run> element: $XmlPath"
    }

    $total = [int]$run.total
    $passed = [int]$run.passed
    $failed = [int]$run.failed
    $skipped = [int]$run.skipped
    $inconclusive = [int]$run.inconclusive
    $executed = $total - $skipped - $inconclusive

    $problems = @()
    if ($total -eq 0) {
        $problems += 'no tests were discovered or executed'
    }
    elseif ($executed -le 0) {
        $problems += "every test was skipped ($skipped of $total)"
    }

    return [pscustomobject]@{
        Total        = $total
        Passed       = $passed
        Failed       = $failed
        Skipped      = $skipped
        Inconclusive = $inconclusive
        Executed     = $executed
        Problems     = $problems
        FailedCases  = @($document.SelectNodes("//test-case[@result='Failed']") | ForEach-Object { $_.name })
    }
}
