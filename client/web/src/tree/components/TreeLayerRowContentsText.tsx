import { HTMLAttributes } from 'react'
import * as React from 'react'

import classNames from 'classnames'

import styles from './TreeLayerRowContentsText.module.scss'

type TreeLayerRowContentsTextProps = HTMLAttributes<HTMLDivElement>

export const TreeLayerRowContentsText: React.FunctionComponent<TreeLayerRowContentsTextProps> = ({
    className,
    children,
    ...rest
}) => (
    <div className={classNames(styles.treeRowContentsText, className)} {...rest}>
        {children}
    </div>
)
