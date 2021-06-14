<#
  .SYNOPSIS
    Script for enabling non powershell src's to execute cmds in Powershell and
    always return compressed json.

  .DESCRIPTION
    Script for handling the execution of powershell cmds from non Powershell
    applications. This script has zero non native dependancies and attempts to
    opperate in as simple and resilent a manner as possible to maximize the
    likelyhood that all errors will occure only in the try statement.

    This script will output a cmd_info object as a compressed json string. It
    will convert the entirety of the object returned from the cmd run to json
    to the maximum depth possible (100). Ensure that you select only the
    properties desired from the cmd run, else the runtime and output may be
    unnessesarilly large.

  .PARAMETER Command
    A string statement that you wish to have powershell run. This cmd will be
    executed directly using the Invoke-Expression native command. This param is
    mandatory, and will be checked prior to invocation but after script
    execution to ensure failure is caught in the try statement that can convert
    it to json on failure.

  .PARAMETER Property
    The name(s) of properties you would like return from the results of the cmd
    you specified. This will filter the result down to just the properties
    specified here.

  .EXAMPLE
    Invoke Get-ChildItem and return only the names

    PS> Invoke-PowerShell.ps1 -Command 'Get-ChildItem -Path /' -Property 'name'

  .EXAMPLE
    Invoke Get-ChildItem and return the name, fullname, and attributes

    PS> Invoke-PowerShell.ps1 -Command 'Get-ChildItem -Path /' -Property 'name','fullname','attributes'
#>

[cmdletbinding()]
param (
  [String]$Command,

  [String[]]$Property
)

Write-Information 'Define the results obj to store information form invokation'
$invoke_obj = @{
  'Success' = $true
  'Command' = $Command
  'Property' = ''
  'Output' = ''
  'ErrorInfo' = @{
    'Message' = ''
    'Line' = ''
    'PositionMessage' = ''
  }
}

try {
  Write-Information 'Ensuring a cmd is present to execute'
  if ('Command' -notin $PSBoundParameters.keys) {
    throw 'No value was specified for the Command parameter'
  }

  Write-Information 'Invoke command else throw terminating error'
  $invoke_expression_param = @{
    'Command' = $Command
    'InformationAction' = 'Continue'
    'Verbose' = $true
    'ErrorAction' = 'Stop'
  }
  $cmd_output = Invoke-Expression @invoke_expression_param

  Write-Information 'Checking for properties to filter by'
  if ('Property' -in $PSBoundParameters.keys) {
    $invoke_obj.Property = $Property
    $cmd_output = $cmd_output | Select-Object -Property $Property
  }

  $invoke_obj.Output = $cmd_output

  Write-Information 'Convert to JSON else throw terminating error'
  $json = $invoke_obj | ConvertTo-Json -Compress -Depth 100 -ErrorAction Stop

} catch {
  $invoke_obj.Success = $false
  $invoke_obj.ErrorInfo.Message = $_.Exception.Message
  $invoke_obj.ErrorInfo.Line = $_.Exception.CommandInvocation.Line
  $invoke_obj.ErrorInfo.PositionMessage = $_.Exception.CommandInvocation.PositionMessage
  $json = $invoke_obj | ConvertTo-Json -Compress -Depth 100

} finally {
  Write-Information 'Write json result to stdout'
  $json
}