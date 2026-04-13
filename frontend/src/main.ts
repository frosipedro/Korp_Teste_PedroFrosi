import { bootstrapApplication } from '@angular/platform-browser'
import { provideRouter } from '@angular/router'
import {
  provideHttpClient,
  withInterceptorsFromDi,
  HTTP_INTERCEPTORS,
} from '@angular/common/http'
import { provideAnimations } from '@angular/platform-browser/animations'
import { importProvidersFrom } from '@angular/core'
import { MatSnackBarModule } from '@angular/material/snack-bar'
import { MatCardModule } from '@angular/material/card'

import { AppComponent } from './app/app.component'
import { routes } from './app/app.routes'
import { ErrorInterceptor } from './app/core/interceptors/error.interceptor'

bootstrapApplication(AppComponent, {
  providers: [
    provideRouter(routes),
    provideAnimations(),
    provideHttpClient(withInterceptorsFromDi()),
    importProvidersFrom(MatSnackBarModule, MatCardModule),
    {
      provide: HTTP_INTERCEPTORS,
      useClass: ErrorInterceptor,
      multi: true,
    },
  ],
}).catch((err) => console.error(err))
