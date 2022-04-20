import { HTMLAttributes } from 'react'
import * as React from 'react'

import classNames from 'classnames'

import styles from './UserSettingReposContainer.module.scss'

type UserSettingReposContainerProps = HTMLAttributes<HTMLElement>

export const UserSettingReposContainer: React.FunctionComponent<UserSettingReposContainerProps> = ({
    children,
    className,
    ...rest
}) => (
    <div className={classNames(className, styles.userSettingsRepos)} {...rest}>
        {children}
    </div>
)
