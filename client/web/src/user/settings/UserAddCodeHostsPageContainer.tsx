import * as React from 'react'

import { codeHostExternalServices } from '../../components/externalServices/externalServices'

import { UserAddCodeHostsPageProps, UserAddCodeHostsPage } from './codeHosts/UserAddCodeHostsPage'

export interface UserAddCodeHostsPageContainerProps
    extends Omit<UserAddCodeHostsPageProps, 'codeHostExternalServices'> {}

export const UserAddCodeHostsPageContainer: React.FunctionComponent<UserAddCodeHostsPageContainerProps> = props => (
    <UserAddCodeHostsPage
        {...props}
        codeHostExternalServices={{
            github: codeHostExternalServices.github,
            gitlabcom: codeHostExternalServices.gitlabcom,
        }}
    />
)
