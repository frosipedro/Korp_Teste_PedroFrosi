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

@Injectable()
export class ErrorInterceptor implements HttpInterceptor {
  constructor(private snackBar: MatSnackBar) {}

  intercept(
    req: HttpRequest<unknown>,
    next: HttpHandler,
  ): Observable<HttpEvent<unknown>> {
    return next.handle(req).pipe(
      catchError((err: HttpErrorResponse) => {
        const message = this.resolveMessage(err)
        this.snackBar.open(message, 'Fechar', {
          duration: 5000,
          panelClass: ['snack-error'],
        })
        return throwError(() => err)
      }),
    )
  }

  private resolveMessage(err: HttpErrorResponse): string {
    if (err.error?.error) {
      return err.error.error
    }
    switch (err.status) {
      case 400:
        return 'Requisição inválida.'
      case 404:
        return 'Recurso não encontrado.'
      case 409:
        return 'Conflito: verifique os dados e tente novamente.'
      case 502:
        return 'Serviço indisponível. Tente novamente em instantes.'
      default:
        return `Erro inesperado (${err.status}). Tente novamente.`
    }
  }
}
