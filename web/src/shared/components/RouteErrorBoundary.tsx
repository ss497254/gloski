import { useRouteError, isRouteErrorResponse, Link } from 'react-router-dom'
import { AlertCircle, Home, RefreshCw } from 'lucide-react'
import { Button } from '@/ui/button'
import { Card, CardContent } from '@/ui/card'

export function RouteErrorBoundary() {
  const error = useRouteError()

  let title = 'Something went wrong'
  let message = 'An unexpected error occurred'
  let details: string | undefined

  if (isRouteErrorResponse(error)) {
    // This is a response error (404, 500, etc.)
    title = `${error.status} ${error.statusText}`
    message = error.data?.message || 'The page you are looking for does not exist'

    if (error.status === 404) {
      title = 'Page Not Found'
      message = 'The page you are looking for does not exist or has been moved'
    } else if (error.status === 401) {
      title = 'Unauthorized'
      message = 'You do not have permission to access this resource'
    } else if (error.status === 500) {
      title = 'Server Error'
      message = 'An error occurred on the server. Please try again later'
    }
  } else if (error instanceof Error) {
    // This is a JavaScript error
    message = error.message
    details = error.stack
  }

  return (
    <div className="h-screen w-full flex items-center justify-center p-4 bg-muted/30">
      <Card className="max-w-lg w-full">
        <CardContent className="p-8">
          <div className="flex flex-col items-center text-center space-y-4">
            {/* Icon */}
            <div className="h-16 w-16 rounded-full bg-destructive/10 flex items-center justify-center">
              <AlertCircle className="h-8 w-8 text-destructive" />
            </div>

            {/* Title */}
            <div className="space-y-2">
              <h1 className="text-2xl font-semibold">{title}</h1>
              <p className="text-muted-foreground">{message}</p>
            </div>

            {/* Error details (development only) */}
            {details && import.meta.env.DEV && (
              <details className="w-full text-left">
                <summary className="cursor-pointer text-sm text-muted-foreground hover:text-foreground">
                  Show error details
                </summary>
                <pre className="mt-2 p-4 bg-muted rounded-lg text-xs overflow-auto max-h-40">
                  {details}
                </pre>
              </details>
            )}

            {/* Actions */}
            <div className="flex items-center gap-2 pt-4">
              <Button variant="outline" onClick={() => window.location.reload()}>
                <RefreshCw className="h-4 w-4 mr-2" />
                Reload Page
              </Button>
              <Button asChild>
                <Link to="/">
                  <Home className="h-4 w-4 mr-2" />
                  Go Home
                </Link>
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
