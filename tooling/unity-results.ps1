# Shared NUnit result evaluation for the Unity test wrappers.
#
# Dot-source this file. A run is only acceptable when it executed at least one
# test and every reported counter is internally consistent: an empty file, a
# fully skipped suite or contradictory counters must never be reported as a
# passing platform.

function Get-UnityTestSummary {
    param([Parameter(Mandatory = $true)][string]$XmlPath)

    if (-not (Test-Path $XmlPath)) {
        throw "result file not found: $XmlPath"
    }

    [xml]$document = Get-Content -Path $XmlPath -ErrorAction Stop
    $run = $document.'test-run'
    if ($null -eq $run) {
        throw "result file has no <test-run> element: $XmlPath"
    }

    $counters = @{}
    foreach ($name in @('total', 'passed', 'failed', 'skipped', 'inconclusive')) {
        $raw = $run.GetAttribute($name)
        if ([string]::IsNullOrWhiteSpace($raw)) {
            $counters[$name] = $null
            continue
        }
        $parsed = 0
        if (-not [int]::TryParse($raw, [ref]$parsed) -or $parsed -lt 0) {
            $counters[$name] = $null
            continue
        }
        $counters[$name] = $parsed
    }

    $problems = @()
    foreach ($name in @('total', 'passed', 'failed', 'skipped', 'inconclusive')) {
        if ($null -eq $counters[$name]) {
            $problems += "missing or non-integer '$name' counter"
        }
    }

    $total = if ($null -eq $counters['total']) { 0 } else { $counters['total'] }
    $passed = if ($null -eq $counters['passed']) { 0 } else { $counters['passed'] }
    $failed = if ($null -eq $counters['failed']) { 0 } else { $counters['failed'] }
    $skipped = if ($null -eq $counters['skipped']) { 0 } else { $counters['skipped'] }
    $inconclusive = if ($null -eq $counters['inconclusive']) { 0 } else { $counters['inconclusive'] }
    $executed = $total - $skipped - $inconclusive

    if ($problems.Count -eq 0) {
        if ($total -eq 0) {
            $problems += 'no tests were discovered or executed'
        }
        elseif ($total -ne ($passed + $failed + $skipped + $inconclusive)) {
            $problems += "inconsistent counters: total=$total but passed+failed+skipped+inconclusive=$($passed + $failed + $skipped + $inconclusive)"
        }
        elseif ($executed -le 0) {
            $problems += "no test was actually executed ($skipped skipped, $inconclusive inconclusive of $total)"
        }
        elseif ($passed -eq 0 -and $failed -eq 0) {
            $problems += "no passing or failing test reported (total=$total)"
        }
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
