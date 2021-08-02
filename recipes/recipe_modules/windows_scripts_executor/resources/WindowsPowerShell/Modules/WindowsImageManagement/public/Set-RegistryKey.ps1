Function Set-RegistryKey {
  <#
    .SYNOPSIS
      Set the value of registry keys, properties, and values.

    .DESCRIPTION
      This is a wrapper function around some of the common
      [Microsoft.Win32.RegistryValue] class methods for getting and setting
      a registry key and its properties. When working with the windows
      registry you have to be sure a key or property is present before taking
      action on it. The commands will not automatically create missing keys or
      properties.

      This function will check for the existence of a key and create it if not
      present. Then it will perform all operations on that object to remove the
      chance of race conditions between the read and write operations.

    .PARAMETER Path
      The registry key path

    .PARAMETER PropertyName
      The name of the registry key property

    .PARAMETER PropertyValue
      The Value of the registry key property

    .PARAMETER PropertyType
      The data type for the registry key property
  #>

  [cmdletbinding(DefaultParameterSetName='none')]
  param (
    [Parameter(Mandatory)]
    [ValidateScript({Test-Path -Path $_ -IsValid})]
    [String]$Path,

    [Parameter(
      Mandatory,
      ParameterSetName='SetProperty')]
    [String]$PropertyName,

    [Parameter(
      Mandatory,
      ParameterSetName='SetProperty')]
    $PropertyValue,

    [Parameter(
      Mandatory,
      ParameterSetName='SetProperty')]
    [Microsoft.Win32.RegistryValueKind]$PropertyType
  )

  Write-Information "Testing for Registry key $Path"
  if (-not (Test-Path $Path)) {
    Write-Information "Creating registry key: $Path"
    $reg_key = New-Item -Path $Path -ErrorAction Stop -Force
  }

  if ($PSCmdlet.ParameterSetName -ne 'SetProperty') {
    return
  }

  if (-not $reg_key) {
    [Microsoft.Win32.RegistryKey]$reg_key = Get-Item -Path $Path -ErrorAction Stop
  }

  Write-Information "Setting $PropertyName to $PropertyValue"
  [Microsoft.Win32.Registry]::SetValue($reg_key.name, $PropertyName, $PropertyValue, $PropertyType)
}
