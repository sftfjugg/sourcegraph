import { forwardRef } from 'react'

import classNames from 'classnames'

import { ForwardReferenceComponent } from '../../..'

import styles from './CardText.module.scss'

interface CardTextProps {}

export const CardText = forwardRef(({ as: Component = 'p', children, className, ...attributes }, reference) => (
    <Component ref={reference} className={classNames(styles.cardText, className)} {...attributes}>
        {children}
    </Component>
)) as ForwardReferenceComponent<'p', CardTextProps>
