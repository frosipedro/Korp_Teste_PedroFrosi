import { Injectable } from '@angular/core'
import {
  HttpInterceptor,
  HttpRequest,
  HttpHandler,
  HttpEvent,
  HttpErrorResponse,
} from '@angular/common/http'
import { Observable, throwError, timer } from 'rxjs'
import { catchError, retry } from 'rxjs/operators'
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
    const shouldRetryTransientErrors = req.method === 'GET'

    return next.handle(req).pipe(
      retry({
        count: shouldRetryTransientErrors ? 3 : 0,
        delay: (error: unknown, retryCount: number) => {
          if (
            !(error instanceof HttpErrorResponse) ||
            !this.isTransientFailure(error)
          ) {
            return throwError(() => error)
          }

          return timer(retryCount * 500)
        },
      }),
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

  private isTransientFailure(error: HttpErrorResponse): boolean {
    return [0, 502, 503, 504].includes(error.status)
  }
}
