import { HTMLAttributes } from 'react'
import * as React from 'react'

import classNames from 'classnames'

import { Tabs, TabsProps } from '@sourcegraph/wildcard'

import styles from './ConnectionPopover.module.scss'

type ConnectionPopoverProps = HTMLAttributes<HTMLDivElement>

export const ConnectionPopover: React.FunctionComponent<ConnectionPopoverProps> = ({
    className,
    children,
    ...rest
}) => (
    <div className={classNames(styles.connectionPopover, className)} {...rest}>
        {children}
    </div>
)

type ConnectionPopoverTabsProps = TabsProps & {
    className?: string
}

export const ConnectionPopoverTabs: React.FunctionComponent<ConnectionPopoverTabsProps> = ({
    children,
    className,
    ...rest
}) => (
    <div className={classNames(styles.connectionPopover, className)}>
        <Tabs {...rest}>{children}</Tabs>
    </div>
)
