import { HttpContextToken } from '@angular/common/http'

export const SUPPRESS_GLOBAL_ERROR_SNACKBAR = new HttpContextToken<boolean>(
  () => false,
)
