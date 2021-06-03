function Close_WindowsImageEnvironment {
  [cmdletbinding()]
  param (
    [WindowsImageInfo]$WindowsImageInfoObj
  )

  if ($WindowsImageInfoObj.Mount) {
    if (Test-Path $WindowsImageInfoObj.Mount.Path) {
      Dismount-WindowsImage -Path $WindowsImageInfoObj.Mount.Path -Save | Out-Null
    }

    $still_mounted = $WindowsImageInfoObj.Mount.Path -in (Get-WindowsImage -Mounted).Path
    if ($still_mounted) {
      Write-Warning "Failed to dismount $($WindowsImageInfoObj.File) from $($WindowsImageInfoObj.Mount.Path)"
    }
    else {
      $WindowsImageInfoObj.Mount = $null
    }
  }

  if ($WindowsImageInfoObj.ContentLocation) {
    if (Test-Path $WindowsImageInfoObj.ContentLocation) {
      Remove-Item -Path $WindowsImageInfoObj.ContentLocation -Recurse -Force -Confirm:$false
    }

    if (Test-Path $WindowsImageInfoObj.ContentLocation) {
      Write-Warning "Failed to remove item $($WindowsImageInfoObj.ContentLocation)"
    }
    else {
      $WindowsImageInfoObj.ContentLocation = $null
    }
  }
  $WindowsImageInfoObj | Write-Output
}