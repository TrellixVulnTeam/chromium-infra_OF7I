function Test_OscdimgInPath {
  [cmdletbinding()]
  Param (
    [String]$Name
  )

  $found_oscdimg = ($env:Path.split(';') | Where-Object {Join-Path $_ $Name | Test-Path}) -ne $null
  if ($found_oscdimg) {
    Write-Information "Found $Name in our `$env:Path"
  }
  else {
    Write-Warning "$Name was not found in `$env:Path."
    Write-Information 'Attempting to initialize Windows Assessment and Deployment kit.'
    Initialize-WindowsAssessmentAndDeploymentKit
  }

  $found_oscdimg = ($env:Path.split(';') | Where-Object {Join-Path $_ $Name | Test-Path}) -ne $null
  if (-not $found_oscdimg) {
    throw "$Name was still not found in `$env:Path after attempted Initialization."
  }
}