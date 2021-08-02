function Edit-OfflineRegistry {
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

      PS> $params = @{
        'OfflineImagePath' = "$env:UserProfile\Desktop\Winpe\mnt"
        'RegistryKeyPath' = "Microsoft\Windows Defender\Features"
        'PropertyName' = 'TamperProtection'
        'PropertyValue' = 0
        'PropertyType' = 'DWord'
      }

      PS> Edit-OfflineRegistry @params
  #>

  [cmdletbinding()]
  param (
    [Parameter(Mandatory)]
    [String]$OfflineImagePath,

    [String]$OfflineRegHiveFile = 'Windows\System32\Config\software',

    [Parameter(Mandatory)]
    [ValidateScript({Test-Path -Path $_ -IsValid})]
    [String]$RegistryKeyPath,

    [Parameter(Mandatory)]
    [String]$PropertyName,

    [Parameter(Mandatory)]
    $PropertyValue,

    [Parameter(Mandatory)]
    [Microsoft.Win32.RegistryValueKind]$PropertyType,

    [Int]$RetryHiveOps = 10,

    [Switch]$SkipUnload
  )

  $offline_hklm_software_file = Join-Path $OfflineImagePath $OfflineHklmSoftwareFile
  $temp_hive = "hklm:\$(New-GUID)"
  $formatted_temp_hive = $temp_hive.replace(':','')

  #region Load Offline Registry
  Write-Information 'Loading offline image Local Machine registry' +
    "$offline_hklm_software_file at $formatted_temp_hive"

  $arg_list = @(
    'load',
    $formatted_temp_hive,
    $offline_hklm_software_file
  )

  $start_process_load_params = @{
    'FilePath' = 'reg.exe'
    'ArgumentList' = $arg_list
    'Wait' = $true
    'InformationAction' = $InformationPreference
    'Verbose' = $VerbosePreference
  }

  for ($i=1; $i -le $RetryHiveOps; $i++) {
    Start-Process @start_process_load_params
    Start-Sleep -Seconds 1

    if (Test-Path -Path $temp_hive) {
      Write-Information "Successfuly loaded offline image Local Machine registry at $temp_hive"
      break
    } elseif ($i -eq $RetryHiveOps) {
      throw 'Failed to load offline OS Local Machine registry'
    }
    else {
      Write-Warning "Failed to load offline image Local Machine registry. Retrying ($i/$RetryHiveOps)"
    }
  }
  #endregion Load Offline Registry

  $path = Join-Path $temp_hive $RegistryKeyPath
  $set_reg_key_params = @{
    'Path'          = $path
    'PropertyName'  = $PropertyName
    'PropertyValue' = $PropertyValue
    'PropertyType'  = $PropertyType
  }
  Set-RegistryKey @set_reg_key_params

  #region Unload Offline Registry
  if (-not $SkipUnload) {
    Write-Information "Attempting to unload offline OS Local Machine Registry at $temp_hive"
    For ($i = 1; $i -le $RetryHiveOps; $i++) {
      Write-Information 'Perform Garbage Collection to remove handles from hive'
      [gc]::Collect()
      $start_process__unload_params = @{
        'FilePath' = 'reg.exe'
        'ArgumentList' = @('unload', $formatted_temp_hive)
        'Wait' = $true
      }
      Start-Process @start_process__unload_params
      Start-Sleep -Seconds 1

      if (Test-Path $temp_hive) {
        Write-Warning "Failed to unload offline OS Local Machine registry. Retrying ($i/$RetryHiveOps)"
      } elseif ($i -eq $RetryHiveOps) {
        throw 'Failed to unload offline OS Local Machine registry'
      }
      else {
        Write-Information 'offline OS Local Machine registry unloaded.'
        break
      }
    }
  } else {
    Write-Information 'Skipping registry unload'
  }
  #endregion Unload Offline Registry
}