import React from 'react'
import { Redirect } from 'react-router'

import { lazyComponent } from '@sourcegraph/shared/src/util/lazyComponent'

import { namespaceAreaRoutes } from '../../namespaces/routes'

import { OrgAreaRoute } from './OrgArea'

const OrgSettingsArea = lazyComponent(() => import('../settings/OrgSettingsArea'), 'OrgSettingsArea')
const OrgMembersArea = lazyComponent(() => import('../members/OrgMembersArea'), 'OrgMembersArea')

const redirectToOrganizationProfile: OrgAreaRoute['render'] = props => (
    <Redirect to={`${props.match.url}/settings/profile`} />
)

export const orgAreaRoutes: readonly OrgAreaRoute[] = [
    {
        path: '/settings',
        render: props => <OrgSettingsArea {...props} isLightTheme={props.isLightTheme} />,
    },
    // PIROS: FFLAG HERE
    {
        path: '/members-v2',
        render: props => <OrgMembersArea {...props} isLightTheme={props.isLightTheme} />,
    },
    ...namespaceAreaRoutes,

    // Redirect from /organizations/:orgname -> /organizations/:orgname/settings/profile.
    {
        path: '/',
        exact: true,
        render: redirectToOrganizationProfile,
    },
    // Redirect from previous /organizations/:orgname/account -> /organizations/:orgname/settings/profile.
    {
        path: '/account',
        render: redirectToOrganizationProfile,
    },
]
