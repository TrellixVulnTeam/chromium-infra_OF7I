class DiskImageInfo {
  # Properties
  # TODO(actodd): Find way to validate the type of CimInstance object.
  # PowerShell classes do not allow the use of ValidateScript on class
  # properties and do not allow overriding the default property getters and
  # setters.
  [System.IO.FileInfo]$File

  [System.IO.FileInfo]$ContentLocation

  [Microsoft.Management.Infrastructure.CimInstance]$Mount

  [System.IO.FileInfo]$DiskRoot

  [System.Management.ManagementObject]$Volume

  # TODO(actodd): Add support for holding multiple images. Due to other
  # assumptions made by the WindowsImageManagement module this attribute needs
  # to be singular instead of holding an array of images. Considering that it
  # is normal for ISO's and WIMs to contain multiple images our object
  # structure and the logic that works with it should support this. At the time
  # of writting this, this feature is out of scope.
  [WindowsImageInfo]$WindowsImage

  # Constructors
  DiskImageInfo (){}

  DiskImageInfo (
    [System.IO.FileInfo]$File
  ){
    foreach ($param in $PSBoundParameters.Keys){
      $this.$param = $PSBoundParameters.$param
    }
  }

  DiskImageInfo (
    [System.IO.FileInfo]$File,
    [System.IO.FileInfo]$ContentLocation
  ){
    foreach ($param in $PSBoundParameters.Keys){
      $this.$param = $PSBoundParameters.$param
    }
  }

  # TODO(actodd): Add override with no params that checks if the values of the
  # class instance properties are still valid and if not sets them to null.
  [void] Update (
    [DiskImageInfo]$DiskImageInfoObj
  )
  {
    $obj = $DiskImageInfoObj
    $properties = ($obj | Get-Member -MemberType Properties).name
    for ($i = 0; $i -lt $properties.count; $i++) {
      $propert_value = $obj | Select-Object -ExpandProperty $properties[$i]
      if ($propert_value) {
        $this.($properties[$i]) = $propert_value
      }
    }
  }
}