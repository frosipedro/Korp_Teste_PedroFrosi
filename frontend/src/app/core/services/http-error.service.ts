import { Injectable } from '@angular/core'
import { HttpErrorResponse } from '@angular/common/http'

export interface StockConflictDetails {
  productId: number
  requested: number
  available: number
}

@Injectable({ providedIn: 'root' })
export class HttpErrorService {
  getMessage(error: unknown): string {
    if (!(error instanceof HttpErrorResponse)) {
      return 'Erro inesperado. Tente novamente.'
    }

    const apiMessage = this.getApiMessage(error)
    if (apiMessage) {
      const translatedMessage = this.translateDomainMessage(apiMessage)
      if (translatedMessage) {
        return translatedMessage
      }

      const stockConflict = this.extractStockConflictDetails(error)
      if (stockConflict) {
        return this.formatStockConflictMessage(stockConflict)
      }

      if (/invoice already closed/i.test(apiMessage)) {
        return 'A nota fiscal já está fechada.'
      }

      if (/invoice not found/i.test(apiMessage)) {
        return 'Nota fiscal não encontrada.'
      }

      if (this.looksLikeValidationError(apiMessage)) {
        return 'Verifique os campos informados e tente novamente.'
      }

      return this.capitalize(apiMessage)
    }

    switch (error.status) {
      case 0:
        return 'Não foi possível conectar ao servidor. Verifique a rede e tente novamente.'
      case 400:
        return 'Verifique os campos informados e tente novamente.'
      case 404:
        return 'Recurso não encontrado.'
      case 409:
        return 'Conflito de dados. Revise as informações e tente novamente.'
      case 502:
        return 'Serviço indisponível. Tente novamente em instantes.'
      default:
        return `Erro inesperado (${error.status}). Tente novamente.`
    }
  }

  extractStockConflictDetails(error: unknown): StockConflictDetails | null {
    if (!(error instanceof HttpErrorResponse)) {
      return null
    }

    const apiMessage = this.getApiMessage(error)
    if (!apiMessage) {
      return null
    }

    const match = apiMessage.match(
      /insufficient stock for product ID (\d+): requested (\d+), available (\d+)/i,
    )
    if (!match) {
      return null
    }

    return {
      productId: Number(match[1]),
      requested: Number(match[2]),
      available: Number(match[3]),
    }
  }

  private translateDomainMessage(message: string): string | null {
    if (
      /context deadline exceeded|client\.timeout exceeded|timeout|deadline exceeded/i.test(
        message,
      )
    ) {
      return 'O serviço de estoque demorou para responder. Tente novamente.'
    }

    if (
      /failed to deduct stock after \d+ attempts: inventory error: product not found/i.test(
        message,
      ) ||
      /inventory error: product not found/i.test(message) ||
      (/failed to validate stock/i.test(message) &&
        /inventory returned 404/i.test(message)) ||
      /inventory returned 404/i.test(message) ||
      /product not found/i.test(message)
    ) {
      return 'Não foi possível concluir a nota porque um dos produtos não existe mais. Atualize a lista e tente novamente.'
    }

    if (
      /failed to deduct stock after \d+ attempts: inventory error: insufficient stock/i.test(
        message,
      ) ||
      /inventory error: insufficient stock/i.test(message) ||
      /insufficient stock/i.test(message)
    ) {
      return 'Estoque insuficiente para concluir a operação. Revise as quantidades e tente novamente.'
    }

    if (/concurrent update detected/i.test(message)) {
      return 'O estoque foi atualizado por outro processo. Tente novamente.'
    }

    if (/failed to close invoice/i.test(message)) {
      return 'Não foi possível fechar a nota fiscal.'
    }

    if (/failed to deduct stock after \d+ attempts/i.test(message)) {
      return 'Não foi possível baixar o estoque da nota. Tente novamente.'
    }

    if (/inventory error:/i.test(message)) {
      return 'Não foi possível validar o estoque. Atualize os dados e tente novamente.'
    }

    return null
  }

  private getApiMessage(error: HttpErrorResponse): string | null {
    const body = error.error

    if (typeof body === 'string') {
      return body.trim() || null
    }

    if (body && typeof body === 'object') {
      const nestedBody = body as Record<string, unknown>
      const nestedMessage = this.readString(nestedBody['error'])
      if (nestedMessage) {
        return nestedMessage
      }

      const message = this.readString(nestedBody['message'])
      if (message) {
        return message
      }
    }

    return null
  }

  private readString(value: unknown): string | null {
    if (typeof value !== 'string') {
      return null
    }

    const trimmed = value.trim()
    return trimmed.length > 0 ? trimmed : null
  }

  private looksLikeValidationError(message: string): boolean {
    return /required|min|max|binding|validation|Key:/i.test(message)
  }

  private capitalize(message: string): string {
    const trimmed = message.trim()
    if (!trimmed) {
      return 'Erro inesperado. Tente novamente.'
    }

    return trimmed.charAt(0).toUpperCase() + trimmed.slice(1)
  }

  private formatStockConflictMessage(
    details: StockConflictDetails | null,
  ): string {
    if (!details) {
      return 'Estoque insuficiente para o item informado.'
    }

    return `Estoque insuficiente para o produto ID ${details.productId}: solicitado ${details.requested}, disponível ${details.available}.`
  }
}
