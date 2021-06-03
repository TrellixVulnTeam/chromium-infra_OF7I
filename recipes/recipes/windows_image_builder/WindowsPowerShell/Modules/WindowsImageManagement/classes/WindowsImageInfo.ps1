class WindowsImageInfo {
  # Properties
  [System.IO.FileInfo]$File

  [System.IO.FileInfo]$ContentLocation

  [Microsoft.Dism.Commands.ImageObject]$Mount

  [ValidateSet('install', 'boot')]
  [String]$ImageType

  [Microsoft.Dism.Commands.WimImageInfoObject]$ImageInfo

  # Constructors
  WindowsImageInfo (){}

  WindowsImageInfo (
    [System.IO.FileInfo]$File
  ){
    foreach ($param in $PSBoundParameters.Keys){
      $this.$param = $PSBoundParameters.$param
    }
  }

  # TODO(actodd): inherit this method instead of copying it from DiskImageInfo
  Update (
    [WindowsImageInfo]$WindowsImageInfoObj
  )
  {
    $obj = $WindowsImageInfoObj
    $properties = ($obj | Get-Member -MemberType Properties).name
    for ($i = 0; $i -lt $properties.count; $i++) {
      $propert_value = $obj | Select-Object -ExpandProperty $properties[$i]
      if ($propert_value) {
        $this.($properties[$i]) = $propert_value
      }
    }
  }
}