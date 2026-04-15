import { Injectable } from '@angular/core'
import { HttpErrorResponse } from '@angular/common/http'

export interface StockConflictDetails {
  productId: number
  requested: number
  available: number
}

type UpstreamService = 'billing' | 'inventory' | null

@Injectable({ providedIn: 'root' })
export class HttpErrorService {
  getMessage(error: unknown): string {
    if (!(error instanceof HttpErrorResponse)) {
      return 'Erro inesperado. Tente novamente.'
    }

    const apiCode = this.getApiCode(error)
    if (apiCode) {
      const codeMessage = this.translateErrorCode(apiCode)
      if (codeMessage) {
        return codeMessage
      }
    }

    const serviceUnavailableMessage = this.getServiceUnavailableMessage(error)
    if (serviceUnavailableMessage) {
      return serviceUnavailableMessage
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
      case 503:
      case 504:
        return 'Serviço indisponível. Tente novamente em instantes.'
      default:
        return `Erro inesperado (${error.status}). Tente novamente.`
    }
  }

  extractStockConflictDetails(error: unknown): StockConflictDetails | null {
    if (!(error instanceof HttpErrorResponse)) {
      return null
    }

    const apiMessage = this.getApiTechnicalMessage(error)
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
    if (/billing service unavailable/i.test(message)) {
      return this.buildServiceUnavailableMessage('billing')
    }

    if (/inventory service unavailable/i.test(message)) {
      return this.buildServiceUnavailableMessage('inventory')
    }

    if (this.looksLikeUpstreamFailure(message)) {
      return this.buildServiceUnavailableMessage(null)
    }

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

    if (
      /product code already exists/i.test(message) ||
      /duplicate key value violates unique constraint .*products_code_key/i.test(
        message,
      ) ||
      /já existe um produto com esse código/i.test(message)
    ) {
      return 'Já existe um produto com esse código.'
    }

    if (/failed to analyze invoice/i.test(message)) {
      return 'Não foi possível gerar a análise por IA. Tente novamente.'
    }

    if (/ai_analysis: GROQ_API_KEY not configured/i.test(message)) {
      return 'A análise por IA não está configurada neste ambiente.'
    }

    if (/ai_analysis:.*(timeout|deadline exceeded)/i.test(message)) {
      return 'A IA demorou para responder. Tente novamente.'
    }

    if (/ai_analysis:/i.test(message)) {
      return 'Não foi possível gerar a análise por IA. Tente novamente.'
    }

    if (/failed to deduct stock after \d+ attempts/i.test(message)) {
      return 'Não foi possível baixar o estoque da nota. Tente novamente.'
    }

    if (/failed to validate stock/i.test(message)) {
      return 'Não foi possível validar o estoque agora. Tente novamente em instantes.'
    }

    if (/failed to deduct stock/i.test(message)) {
      return 'Não foi possível baixar o estoque da nota. Tente novamente em instantes.'
    }

    if (/inventory error:/i.test(message)) {
      return 'Não foi possível validar o estoque. Atualize os dados e tente novamente.'
    }

    return null
  }

  private translateErrorCode(code: string): string | null {
    switch (code) {
      case 'BILLING_UNAVAILABLE':
        return this.buildServiceUnavailableMessage('billing')
      case 'INVENTORY_UNAVAILABLE':
        return this.buildServiceUnavailableMessage('inventory')
      default:
        return null
    }
  }

  private getServiceUnavailableMessage(
    error: HttpErrorResponse,
  ): string | null {
    const technicalMessage = this.getApiTechnicalMessage(error)
    const unavailableStatus = [0, 502, 503, 504].includes(error.status)

    if (!unavailableStatus && !technicalMessage) {
      return null
    }

    if (technicalMessage && !this.looksLikeUpstreamFailure(technicalMessage)) {
      return null
    }

    return this.buildServiceUnavailableMessage(
      this.detectService(error, technicalMessage ?? ''),
    )
  }

  private getApiCode(error: HttpErrorResponse): string | null {
    const body = error.error
    if (!body || typeof body !== 'object') {
      return null
    }

    const code = this.readString((body as Record<string, unknown>)['code'])
    return code ? code.toUpperCase() : null
  }

  private getApiMessage(error: HttpErrorResponse): string | null {
    const body = error.error

    if (typeof body === 'string') {
      return this.normalizeApiText(body)
    }

    if (body && typeof body === 'object') {
      const nestedBody = body as Record<string, unknown>
      const message = this.normalizeApiText(
        this.readString(nestedBody['message']),
      )
      if (message) {
        return message
      }

      const nestedMessage = this.normalizeApiText(
        this.readString(nestedBody['error']),
      )
      if (nestedMessage) {
        return nestedMessage
      }
    }

    return null
  }

  private getApiTechnicalMessage(error: HttpErrorResponse): string | null {
    const body = error.error
    if (body && typeof body === 'object') {
      const nestedBody = body as Record<string, unknown>
      const technicalError = this.normalizeApiText(
        this.readString(nestedBody['error']),
      )
      if (technicalError) {
        return technicalError
      }
    }

    return this.getApiMessage(error)
  }

  private detectService(
    error: HttpErrorResponse,
    text: string,
  ): UpstreamService {
    const haystack = `${error.url ?? ''} ${text}`.toLowerCase()
    if (
      haystack.includes('/api/billing') ||
      haystack.includes('billing:8082') ||
      haystack.includes('billing service')
    ) {
      return 'billing'
    }

    if (
      haystack.includes('/api/inventory') ||
      haystack.includes('inventory:8081') ||
      haystack.includes('inventory service')
    ) {
      return 'inventory'
    }

    return null
  }

  private looksLikeUpstreamFailure(message: string): boolean {
    return (
      /upstream|bad gateway|gateway timeout|service unavailable|temporarily unavailable/i.test(
        message,
      ) ||
      /connection refused|dial tcp|connect\(\) failed|no live upstreams|econnrefused/i.test(
        message,
      )
    )
  }

  private buildServiceUnavailableMessage(service: UpstreamService): string {
    switch (service) {
      case 'billing':
        return 'O serviço de faturamento está temporariamente indisponível. Tente novamente em instantes.'
      case 'inventory':
        return 'O serviço de estoque está temporariamente indisponível. Tente novamente em instantes.'
      default:
        return 'Um serviço essencial está indisponível no momento. Tente novamente em instantes.'
    }
  }

  private normalizeApiText(value: string | null): string | null {
    if (!value) {
      return null
    }

    const trimmed = value.trim()
    if (!trimmed || this.looksLikeHtml(trimmed)) {
      return null
    }

    return trimmed
  }

  private looksLikeHtml(value: string): boolean {
    return /<!doctype html|<html|<head>|<body>|<title>|<\/\w+>/i.test(value)
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
