import { Routes } from '@angular/router'

export const routes: Routes = [
  {
    path: '',
    redirectTo: 'products',
    pathMatch: 'full',
  },
  {
    path: 'products',
    loadComponent: () =>
      import('./features/products/product-list.component').then(
        (m) => m.ProductListComponent,
      ),
  },
  {
    path: 'products/new',
    loadComponent: () =>
      import('./features/products/product-form.component').then(
        (m) => m.ProductFormComponent,
      ),
  },
  {
    path: 'products/edit/:id',
    loadComponent: () =>
      import('./features/products/product-form.component').then(
        (m) => m.ProductFormComponent,
      ),
  },
  {
    path: 'invoices',
    loadComponent: () =>
      import('./features/invoices/invoice-list.component').then(
        (m) => m.InvoiceListComponent,
      ),
  },
  {
    path: 'invoices/new',
    loadComponent: () =>
      import('./features/invoices/invoice-form.component').then(
        (m) => m.InvoiceFormComponent,
      ),
  },
  {
    path: '**',
    redirectTo: 'products',
  },
]
