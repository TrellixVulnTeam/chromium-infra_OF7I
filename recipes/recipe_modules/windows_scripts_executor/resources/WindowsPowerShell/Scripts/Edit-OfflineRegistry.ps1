<#
  .SYNOPSIS
    Modify Offline Registry Hive of Offline Image

  .DESCRIPTION
    Load a registry hive file to a temp location on your running Windows
    hosts. Modify its contents using the Microsoft.Win32.RegistryValue
    class. After this is complete, unload the registry hive.

  .PARAMETER OfflineImagePath
    The path to the root of the offline image containing the offline
    registry file that you want to modify

  .PARAMETER OfflineRegHiveFile
    The path from the root of the offline image to the offline registry
    hive file that you want to load and modify

  .PARAMETER RegistryKeyPath
    The path from the root of the registry to the key that you want to create
    and\or modify

  .PARAMETER PropertyName
    The name of the registry key property that you want to create and\or
    modify

  .PARAMETER PropertyValue
    The Value that you want to set the regitry property too

  .PARAMETER PropertyType
    The type of value you are setting the registry key property too

  .PARAMETER RetryHiveOps
    The number of times to individually retry loading and unloading the
    registry hive file

  .PARAMETER SkipUnload
    Switch to skip unloading the registry. This can be used to debug and
    validate changes

  .EXAMPLE
    Modify the hklm software registry hive of an offline image mounted at ~\Desktop\Win10\mnt

    PS> Edit-OfflineRegistry -OfflineImagePath "$env:UserProfile\Desktop\Winpe\mnt" -RegistryKeyPath "Microsoft\Windows Defender\Features" -PropertyName 'TamperProtection' -PropertyValue 0 -PropertyType 'DWord'
#>

[cmdletbinding()]
param (
  $OfflineImagePath,

  $OfflineRegHiveFile = 'Windows\System32\Config\software',

  $RegistryKeyPath,

  $PropertyName,

  $PropertyValue,

  $PropertyType,

  $RetryHiveOps = 10,

  $SkipUnload
)

$invoke_obj = @{
  'Success' = $true
  'Output' = ''
  'ErrorInfo' = @{
    'Message' = ''
    'Line' = ''
    'PositionMessage' = ''
  }
}

$required_params = @(
  'OfflineImagePath',
  'RegistryKeyPath',
  'PropertyName',
  'PropertyValue',
  'PropertyType',
)

try {
  #region Check for required params
  for ($p in $required_params) {
    if ($p -notin $PSBoundParameters.keys) {
      throw "$p is a required Parameter"
    }
  }
  #endregion Check for required params

  $windows_image_management = "$PSScriptRoot\..\Modules\WindowsImageManagement"
  Import-Module -Name $windows_image_management -ErrorAction Stop | Out-Null

  $params = @{
    'OfflineImagePath' = $OfflineImagePath
    'OfflineRegHiveFile' = $OfflineRegHiveFile
    'RegistryKeyPath' = $RegistryKeyPath
    'PropertyName' = $PropertyName
    'PropertyValue' = $PropertyValue
    'PropertyType' = $PropertyType
    'RetryHiveOps' = $RetryHiveOps
    'SkipUnload' = $RetryHiveOps
    'InformationAction' = $InformationPreference
    'Verbose' = $VerbosePreference
    'ErrorAction' = 'Stop'
  }

  $invoke_obj.Output = Edit-OfflineRegistry @params

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