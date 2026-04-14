import { Injectable } from '@angular/core'
import {
  HttpInterceptor,
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpErrorResponse,
} from '@angular/common/http'
import { Observable, throwError } from 'rxjs'
import { catchError } from 'rxjs/operators'
import { MatSnackBar } from '@angular/material/snack-bar'
import { HttpErrorService } from '../services/http-error.service'
import { SUPPRESS_GLOBAL_ERROR_SNACKBAR } from '../http/request-flags'

@Injectable()
export class ErrorInterceptor implements HttpInterceptor {
  constructor(
    private snackBar: MatSnackBar,
    private httpErrorService: HttpErrorService,
  ) {}

  intercept(
    req: HttpRequest<unknown>,
    next: HttpHandler,
  ): Observable<HttpEvent<unknown>> {
    return next.handle(req).pipe(
      catchError((err: HttpErrorResponse) => {
        if (!req.context.get(SUPPRESS_GLOBAL_ERROR_SNACKBAR)) {
          const message = this.httpErrorService.getMessage(err)
          this.snackBar.open(message, 'Fechar', {
            duration: 5000,
            panelClass: ['snack-error'],
          })
        }
        return throwError(() => err)
      }),
    )
  }
}
