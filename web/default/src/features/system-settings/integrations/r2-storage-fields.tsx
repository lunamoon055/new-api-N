import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import type { MediaStorageProvider } from '../types'

type R2StorageFieldsProps = {
  provider: MediaStorageProvider
  onChange: (patch: Partial<MediaStorageProvider>) => void
}

export function R2StorageFields(props: R2StorageFieldsProps) {
  const { t } = useTranslation()
  const id = useId()
  return (
    <FieldGroup>
      <Field>
        <FieldLabel htmlFor={`${id}-endpoint`}>
          {t('R2 S3 endpoint')}
        </FieldLabel>
        <Input
          id={`${id}-endpoint`}
          type='url'
          value={props.provider.r2_endpoint ?? ''}
          onChange={(event) =>
            props.onChange({ r2_endpoint: event.target.value })
          }
          placeholder='https://<ACCOUNT_ID>.r2.cloudflarestorage.com'
          autoComplete='off'
        />
        <FieldDescription>
          {t(
            'Copy the S3 API endpoint from R2. A trailing bucket name is accepted.'
          )}
        </FieldDescription>
      </Field>
      <Field>
        <FieldLabel htmlFor={`${id}-bucket`}>{t('R2 bucket name')}</FieldLabel>
        <Input
          id={`${id}-bucket`}
          value={props.provider.r2_bucket ?? ''}
          onChange={(event) =>
            props.onChange({ r2_bucket: event.target.value })
          }
          placeholder='my-media-bucket'
          autoComplete='off'
        />
      </Field>
      <Field>
        <FieldLabel htmlFor={`${id}-public`}>
          {t('Public media URL')}
        </FieldLabel>
        <Input
          id={`${id}-public`}
          type='url'
          value={props.provider.r2_public_url ?? ''}
          onChange={(event) =>
            props.onChange({ r2_public_url: event.target.value })
          }
          placeholder='https://media.example.com'
          autoComplete='off'
        />
        <FieldDescription>
          {t(
            'Use the public domain connected to this bucket. The S3 API endpoint cannot serve public media.'
          )}
        </FieldDescription>
      </Field>
      <Field>
        <FieldLabel htmlFor={`${id}-prefix`}>
          {t('Object key prefix')}
        </FieldLabel>
        <Input
          id={`${id}-prefix`}
          value={props.provider.r2_key_prefix ?? ''}
          onChange={(event) =>
            props.onChange({ r2_key_prefix: event.target.value })
          }
          placeholder='new-api/media'
          autoComplete='off'
        />
      </Field>
      <Field>
        <FieldLabel htmlFor={`${id}-access`}>Access Key ID</FieldLabel>
        <Input
          id={`${id}-access`}
          type='password'
          value={props.provider.r2_access_key_id ?? ''}
          onChange={(event) =>
            props.onChange({ r2_access_key_id: event.target.value })
          }
          autoComplete='new-password'
          spellCheck={false}
        />
      </Field>
      <Field>
        <FieldLabel htmlFor={`${id}-secret`}>Secret Access Key</FieldLabel>
        <Input
          id={`${id}-secret`}
          type='password'
          value={props.provider.r2_secret_access_key ?? ''}
          onChange={(event) =>
            props.onChange({ r2_secret_access_key: event.target.value })
          }
          autoComplete='new-password'
          spellCheck={false}
        />
        <FieldDescription>
          {t(
            'Use R2 S3 credentials with object read and write access to this bucket. Keep ******** unchanged to preserve saved credentials. A disabled provider can be saved without credentials.'
          )}
        </FieldDescription>
      </Field>
    </FieldGroup>
  )
}
